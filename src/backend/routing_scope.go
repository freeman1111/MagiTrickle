package magitrickle

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"magitrickle/internal/interfaces"

	"github.com/rs/zerolog/log"
)

const bypassMarksRetryInterval = 15 * time.Second

// setupBypassPolicies получает метки политик из bypassPolicies до включения групп и следит за их изменением
func (a *App) setupBypassPolicies(ctx context.Context) {
	policyNames := slices.Clone(a.config.BypassPolicies)
	if len(policyNames) == 0 {
		return
	}

	marks, missing, err := resolveBypassMarks(policyNames)
	if errors.Is(err, interfaces.ErrPoliciesNotSupported) {
		log.Warn().Err(err).Msg("bypassPolicies is ignored")
		return
	}
	if err != nil {
		log.Warn().Err(err).Msg("failed to get access policy marks, policy traffic will be routed until retry")
	} else {
		err = a.applyBypassMarks(marks, missing)
	}

	a.bypassMarksRefresh = make(chan struct{}, 1)
	go a.watchBypassMarks(ctx, policyNames, err != nil)
}

// requestBypassMarksRefresh просит перечитать метки политик, не блокируя вызывающего
func (a *App) requestBypassMarksRefresh() {
	select {
	case a.bypassMarksRefresh <- struct{}{}:
	default:
	}
}

// watchBypassMarks перечитывает метки по событиям netfilter.d, а пока роутер не отвечает, повторяет попытки
func (a *App) watchBypassMarks(ctx context.Context, policyNames []string, retryNeeded bool) {
	var retry <-chan time.Time
	if retryNeeded {
		retry = time.After(bypassMarksRetryInterval)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.bypassMarksRefresh:
		case <-retry:
		}

		retry = nil
		if err := a.refreshBypassMarks(policyNames); err != nil {
			log.Warn().Err(err).Msg("failed to refresh access policy marks")
			retry = time.After(bypassMarksRetryInterval)
		}
	}
}

// refreshBypassMarks перечитывает метки политик; при ошибке запроса прежние метки остаются в силе
func (a *App) refreshBypassMarks(policyNames []string) error {
	marks, missing, err := resolveBypassMarks(policyNames)
	if err != nil {
		return err
	}
	if slices.Equal(marks, a.nfHelper.BypassMarks.Load()) {
		return nil
	}
	return a.applyBypassMarks(marks, missing)
}

func (a *App) applyBypassMarks(marks []uint32, missing []string) error {
	for _, policyName := range missing {
		log.Warn().Str("policy", policyName).Msg("access policy not found, its traffic will be routed")
	}

	a.nfHelper.BypassMarks.Store(marks)
	log.Info().Int("count", len(marks)).Msg("access policy marks updated")

	var errs []error
	for _, group := range a.ruleSetSnapshot() {
		if err := group.RefreshIPTables(); err != nil {
			errs = append(errs, fmt.Errorf("failed to refresh group %s: %w", group.IDValue().String(), err))
		}
	}
	return errors.Join(errs...)
}

// RefreshIPTables пересобирает iptables-правила включённой группы
func (g *RuleSet) RefreshIPTables() error {
	g.locker.Lock()
	defer g.locker.Unlock()

	if !g.Enabled() || !g.ConfiguredEnabled() || g.ipsetToLink == nil {
		return nil
	}
	return g.ipsetToLink.RefreshIPTablesRules()
}

// resolveBypassMarks возвращает fwmark политик доступа из bypassPolicies одним запросом к роутеру
func resolveBypassMarks(policyNames []string) (marks []uint32, missing []string, err error) {
	if len(policyNames) == 0 {
		return nil, nil, nil
	}

	policyMarks, err := interfaces.GetPolicyMarks()
	if err != nil {
		return nil, nil, err
	}

	marks, missing = pickBypassMarks(policyNames, policyMarks)
	return marks, missing, nil
}

func pickBypassMarks(policyNames []string, policyMarks map[string]uint32) (marks []uint32, missing []string) {
	marks = make([]uint32, 0, len(policyNames))
	for _, policyName := range policyNames {
		mark, ok := policyMarks[policyName]
		if !ok {
			missing = append(missing, policyName)
			continue
		}
		log.Debug().Str("policy", policyName).Int("mark", int(mark)).Msg("bypassing policy traffic")
		marks = append(marks, mark)
	}
	return marks, missing
}
