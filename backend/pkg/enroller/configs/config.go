package configs

import "github.com/kelseyhightower/envconfig"

type Config struct {
	Port string

	UIHost     string
	UIPort     string
	UIProtocol string

	ConsulProtocol string
	ConsulHost     string
	ConsulPort     string

	KeycloakHostname string
	KeycloakPort     string
	KeycloakProtocol string
	KeycloakRealm    string
	KeycloakCA       string

	CertFile     string
	KeyFile      string
	ProxyAddress string
	ProxyCA      string
}

func NewConfig(prefix string) (Config, error) {
	var cfg Config
	err := envconfig.Process(prefix, &cfg)
	if err != nil {
		return Config{}, err
	}
	return cfg, nil
}
