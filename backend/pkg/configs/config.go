package configs

import "github.com/kelseyhightower/envconfig"

type Config struct {
	Port string

	UIHost     string
	UIPort     string
	UIProtocol string

	CAPath string

	CertFile string
	KeyFile  string
}

func NewConfig() (Config, error) {
	var cfg Config
	err := envconfig.Process("manufacturing", &cfg)
	if err != nil {
		return Config{}, err
	}
	return cfg, nil
}
