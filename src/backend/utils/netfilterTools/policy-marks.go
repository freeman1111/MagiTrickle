package netfilterTools

import (
	"errors"
	"sync/atomic"

	"github.com/rs/zerolog/log"
)

// BypassMarks хранит fwmark политик доступа, трафик которых группы не маркируют
type BypassMarks struct {
	marks atomic.Pointer[[]uint32]
}

func (b *BypassMarks) Store(marks []uint32) {
	b.marks.Store(&marks)
}

func (b *BypassMarks) Load() []uint32 {
	if marks := b.marks.Load(); marks != nil {
		return *marks
	}
	return nil
}

// PolicyMarks хранит fwmark политик доступа роутера по их системным именам (Policy0)
type PolicyMarks struct {
	marks atomic.Pointer[map[string]uint32]
}

func (p *PolicyMarks) Store(marks map[string]uint32) {
	p.marks.Store(&marks)
}

func (p *PolicyMarks) Load() map[string]uint32 {
	if marks := p.marks.Load(); marks != nil {
		return *marks
	}
	return nil
}

// Mark возвращает fwmark политики с таким системным именем
func (p *PolicyMarks) Mark(policyID string) (uint32, bool) {
	mark, ok := p.Load()[policyID]
	return mark, ok
}

// enablePolicy направляет группу в политику доступа роутера: пакеты получают метку политики,
// а маршрут выбирает сам роутер по своим ip rule и таблице политики, поэтому свои ip rule и
// маршруты группа не создаёт. Если политики нет, группа включается как обычная связка с
// несуществующим интерфейсом, то есть с blackhole-маршрутом, и трафик не уходит мимо туннеля
func (r *IPSetToLink) enablePolicy() (bool, error) {
	mark, ok := r.nh.Policies.Mark(r.ifaceName)
	r.policy = ok
	if !ok {
		return false, nil
	}

	r.mark, r.table = mark, 0

	if err := r.insertIPTablesRules(r.nh.IPTables4); err != nil {
		return true, err
	}
	if err := r.insertIPTablesRules(r.nh.IPTables6); err != nil {
		return true, err
	}
	return true, nil
}

// RefreshIPTablesRules пересобирает правила включённой связки после изменения политик доступа.
// Если политика цели появилась, исчезла или сменила метку, связка переподключается целиком,
// ipset группы при этом не трогается
func (r *IPSetToLink) RefreshIPTablesRules() error {
	r.locker.Lock()
	defer r.locker.Unlock()

	if !r.enabled.Load() {
		return nil
	}

	if mark, ok := r.nh.Policies.Mark(r.ifaceName); ok != r.policy || (ok && mark != r.mark) {
		log.Info().
			Str("target", r.ifaceName).
			Bool("policy", ok).
			Msg("routing target changed, relinking group")

		disableErr := r.disable()
		if err := r.enable(); err != nil {
			_ = r.disable()
			return errors.Join(disableErr, err)
		}
		return disableErr
	}

	var errs []error
	errs = append(errs, r.insertIPTablesRules(r.nh.IPTables4))
	errs = append(errs, r.insertIPTablesRules(r.nh.IPTables6))
	return errors.Join(errs...)
}
