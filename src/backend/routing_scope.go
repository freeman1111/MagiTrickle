package magitrickle

import (
	"magitrickle/internal/interfaces"

	"github.com/rs/zerolog/log"
)

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
