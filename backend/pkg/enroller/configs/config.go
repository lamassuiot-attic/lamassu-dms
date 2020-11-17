package configs

import "github.com/kelseyhightower/envconfig"

type Config struct {
	Port string

	UIHost     string
	UIPort     string
	UIProtocol string

	PostgresUser     string
	PostgresDB       string
	PostgresPassword string
	PostgresHostname string
	PostgresPort     string

	KeycloakHostname string
	KeycloakPort     string
	KeycloakProtocol string
	KeycloakRealm    string

	CertFile     string
	KeyFile      string
	ProxyAddress string
}

func NewConfig() (Config, error) {
	var cfg Config
	err := envconfig.Process("enroller", &cfg)
	if err != nil {
		return Config{}, err
	}
	return cfg, nil
}
