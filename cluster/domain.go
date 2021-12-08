package cluster

import (
	"github.com/integration-system/isp-kit/json"
)

type AddressConfiguration struct {
	IP   string
	Port string
}

type ConfigData struct {
	Version       string
	Schema        json.RawMessage
	DefaultConfig json.RawMessage
}

type ModuleInfo struct {
	ModuleName       string
	ModuleVersion    string
	GrpcOuterAddress AddressConfiguration
	Endpoints        []EndpointDescriptor
}

type RoutingConfig []BackendDeclaration

type BackendDeclaration struct {
	ModuleName      string
	Version         string
	LibVersion      string
	Endpoints       []EndpointDescriptor
	RequiredModules []ModuleDependency
	Address         AddressConfiguration
}

type EndpointDescriptor struct {
	Path             string
	Inner            bool
	UserAuthRequired bool
	Extra            map[string]interface{}
	Handler          interface{} `json:"-"`
}

type ModuleRequirements struct {
	RequiredModules []string
	RequireRoutes   bool
}

type ModuleDependency struct {
	Name     string
	Required bool
}
