package interfaces

import "errors"

var ErrPoliciesNotSupported = errors.New("access policies are not supported on this platform")

// Policy – политика доступа роутера: системное имя (Policy0), описание из веб-интерфейса и её fwmark
type Policy struct {
	ID          string
	Description string
	Mark        uint32
}

type RouterSpecificAPI interface {
	GetIfaceAliases() (map[string]string, error)
	GetPolicies() ([]Policy, error)
}

var routerAPI RouterSpecificAPI

func init() {
	routerAPI = initRouterSpecificAPI()
}

type DummyRouterSpecificAPI struct{}

func (DummyRouterSpecificAPI) GetIfaceAliases() (map[string]string, error) {
	return map[string]string{}, nil
}

func (DummyRouterSpecificAPI) GetPolicies() ([]Policy, error) {
	return nil, ErrPoliciesNotSupported
}
