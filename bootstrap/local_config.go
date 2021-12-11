package bootstrap

type LocalConfig struct {
	ConfigServiceAddress    ConfigServiceAddr
	GrpcOuterAddress        GrpcOuterAddr
	GrpcInnerAddress        GrpcInnerAddr
	ModuleName              string `valid:"required"`
	AppMode                 string
	DefaultRemoteConfigPath string
	RemoteConfigOverride    string
}

type ConfigServiceAddr struct {
	IP   string `valid:"required"`
	Port string `valid:"required"`
}

type GrpcOuterAddr struct {
	IP   string
	Port int `valid:"required"`
}

type GrpcInnerAddr struct {
	IP   string `valid:"required"`
	Port int    `valid:"required"`
}
