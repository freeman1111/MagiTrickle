package interfaces

import (
	"errors"
	"fmt"
	"net"
	"slices"

	"magitrickle/constant"
	"magitrickle/models"

	"github.com/rs/zerolog/log"
)

func List(showAll bool) ([]models.InterfaceInfo, error) {
	networkInterfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get interfaces: %w", err)
	}

	if !showAll {
		networkInterfaces = filterManaged(networkInterfaces)
	}

	friendlyNames, err := routerAPI.GetIfaceAliases()
	if err != nil {
		log.Debug().Err(err).Msg("failed to load interface aliases")
	}

	interfaces := make([]models.InterfaceInfo, 0, len(networkInterfaces))
	for _, iface := range networkInterfaces {
		interfaces = append(interfaces, models.InterfaceInfo{
			ID:   iface.Name,
			Name: friendlyNames[iface.Name],
		})
	}
	interfaces = append(interfaces, listPolicies()...)

	return interfaces, nil
}

// GetPolicies возвращает политики доступа роутера
func GetPolicies() ([]Policy, error) {
	return routerAPI.GetPolicies()
}

// listPolicies добавляет политики доступа в список целей маршрутизации группы
func listPolicies() []models.InterfaceInfo {
	policies, err := routerAPI.GetPolicies()
	if err != nil {
		if !errors.Is(err, ErrPoliciesNotSupported) {
			log.Debug().Err(err).Msg("failed to load access policies")
		}
		return nil
	}

	result := make([]models.InterfaceInfo, 0, len(policies))
	for _, policy := range policies {
		result = append(result, models.InterfaceInfo{
			ID:   policy.ID,
			Name: policy.Description,
		})
	}
	return result
}

func filterManaged(interfaces []net.Interface) []net.Interface {
	filtered := make([]net.Interface, 0, len(interfaces))
	for _, iface := range interfaces {
		if iface.Flags&net.FlagPointToPoint == 0 || slices.Contains(constant.IgnoredInterfaces, iface.Name) {
			continue
		}
		filtered = append(filtered, iface)
	}

	return filtered
}
