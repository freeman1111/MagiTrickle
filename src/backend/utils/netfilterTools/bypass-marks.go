package netfilterTools

import (
	"errors"
	"sync/atomic"
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

// RefreshIPTablesRules пересобирает правила включённой связки, например после смены меток политик
func (r *IPSetToLink) RefreshIPTablesRules() error {
	r.locker.Lock()
	defer r.locker.Unlock()

	if !r.enabled.Load() {
		return nil
	}

	var errs []error
	errs = append(errs, r.insertIPTablesRules(r.nh.IPTables4))
	errs = append(errs, r.insertIPTablesRules(r.nh.IPTables6))
	return errors.Join(errs...)
}
