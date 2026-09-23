package interfaces

import "errors"

var ErrPoliciesNotSupported = errors.New("access policies are not supported on this platform")

type RouterSpecificAPI interface {
	GetIfaceAliases() (map[string]string, error)
	GetPolicyMark(name string) (uint32, error)
}

var routerAPI RouterSpecificAPI

func init() {
	routerAPI = initRouterSpecificAPI()
}

type DummyRouterSpecificAPI struct{}

func (DummyRouterSpecificAPI) GetIfaceAliases() (map[string]string, error) {
	return map[string]string{}, nil
}

func (DummyRouterSpecificAPI) GetPolicyMark(name string) (uint32, error) {
	return 0, ErrPoliciesNotSupported
}
