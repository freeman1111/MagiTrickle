package magitrickle

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"magitrickle/internal/interfaces"

	"github.com/rs/zerolog/log"
)

const policiesRetryInterval = 15 * time.Second

// setupPolicies читает политики доступа роутера до включения групп и следит за их изменением.
// Политики нужны для bypassPolicies и для групп, направленных в политику вместо интерфейса
func (a *App) setupPolicies(ctx context.Context) {
	bypassNames := slices.Clone(a.config.BypassPolicies)

	policies, err := interfaces.GetPolicies()
	if errors.Is(err, interfaces.ErrPoliciesNotSupported) {
		log.Debug().Err(err).Msg("access policies are disabled")
		return
	}
	if err != nil {
		log.Warn().Err(err).Msg("failed to get access policies, will retry")
	} else {
		err = a.applyPolicies(policies, bypassNames)
	}

	a.policiesRefresh = make(chan struct{}, 1)
	go a.watchPolicies(ctx, bypassNames, err != nil)
}

// requestPoliciesRefresh просит перечитать политики, не блокируя вызывающего
func (a *App) requestPoliciesRefresh() {
	select {
	case a.policiesRefresh <- struct{}{}:
	default:
	}
}

// watchPolicies перечитывает политики по событиям netfilter.d, а пока роутер не отвечает, повторяет попытки
func (a *App) watchPolicies(ctx context.Context, bypassNames []string, retryNeeded bool) {
	var retry <-chan time.Time
	if retryNeeded {
		retry = time.After(policiesRetryInterval)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.policiesRefresh:
		case <-retry:
		}

		retry = nil
		if err := a.refreshPolicies(bypassNames); err != nil {
			log.Warn().Err(err).Msg("failed to refresh access policies")
			retry = time.After(policiesRetryInterval)
		}
	}
}

// refreshPolicies перечитывает политики; при ошибке запроса прежние метки остаются в силе
func (a *App) refreshPolicies(bypassNames []string) error {
	policies, err := interfaces.GetPolicies()
	if err != nil {
		return err
	}
	return a.applyPolicies(policies, bypassNames)
}

func (a *App) applyPolicies(policies []interfaces.Policy, bypassNames []string) error {
	policyMarks := make(map[string]uint32, len(policies))
	for _, policy := range policies {
		policyMarks[policy.ID] = policy.Mark
	}
	bypassMarks, missing := pickBypassMarks(bypassNames, policies)

	current := a.nfHelper.Policies.Load()
	if current != nil && maps.Equal(policyMarks, current) && slices.Equal(bypassMarks, a.nfHelper.BypassMarks.Load()) {
		return nil
	}

	for _, policyName := range missing {
		log.Info().Str("policy", policyName).Msg("access policy from bypassPolicies not found, nothing to bypass")
	}

	a.nfHelper.Policies.Store(policyMarks)
	a.nfHelper.BypassMarks.Store(bypassMarks)
	log.Info().
		Int("policies", len(policyMarks)).
		Int("bypass", len(bypassMarks)).
		Msg("access policies updated")

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

// pickBypassMarks находит метки политик из bypassPolicies по системному имени или описанию без учёта регистра
func pickBypassMarks(policyNames []string, policies []interfaces.Policy) (marks []uint32, missing []string) {
	marks = make([]uint32, 0, len(policyNames))
	for _, policyName := range policyNames {
		idx := slices.IndexFunc(policies, func(policy interfaces.Policy) bool {
			return policy.ID == policyName || strings.EqualFold(policy.Description, policyName)
		})
		if idx < 0 {
			missing = append(missing, policyName)
			continue
		}
		log.Debug().Str("policy", policyName).Int("mark", int(policies[idx].Mark)).Msg("bypassing policy traffic")
		marks = append(marks, policies[idx].Mark)
	}
	return marks, missing
}
