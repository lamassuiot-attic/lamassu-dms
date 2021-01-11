package consul

import (
	"strconv"

	"math/rand"

	"device-manufacturing-system/pkg/enroller/discovery"

	"github.com/go-kit/kit/log"
	consulsd "github.com/go-kit/kit/sd/consul"
	"github.com/hashicorp/consul/api"
)

type ServiceDiscovery struct {
	client    consulsd.Client
	logger    log.Logger
	registrar *consulsd.Registrar
}

func NewServiceDiscovery(consulProtocol string, consulHost string, consulPort string, logger log.Logger) (discovery.Service, error) {
	consulConfig := api.DefaultConfig()
	consulConfig.Address = consulProtocol + "://" + consulHost + ":" + consulPort
	consulClient, err := api.NewClient(consulConfig)
	if err != nil {
		return nil, err
	}
	client := consulsd.NewClient(consulClient)
	return &ServiceDiscovery{client: client, logger: logger}, nil
}

func (sd *ServiceDiscovery) Register(advProtocol string, advHost string, advPort string) error {
	check := api.AgentServiceCheck{
		HTTP:          advProtocol + "://" + advHost + ":" + advPort + "/v1/health",
		Interval:      "10s",
		Timeout:       "1s",
		TLSSkipVerify: true,
		Notes:         "Basic health checks",
	}

	port, _ := strconv.Atoi(advPort)
	num := rand.Intn(100)
	asr := api.AgentServiceRegistration{
		ID:      "manufacturingenroll" + strconv.Itoa(num),
		Name:    "manufacturingenroll",
		Address: advProtocol + "://" + advHost,
		Port:    port,
		Tags:    []string{"manufacturingenroll", "device-manufacturing-system"},
		Check:   &check,
	}
	sd.registrar = consulsd.NewRegistrar(sd.client, &asr, sd.logger)
	sd.registrar.Register()
	return nil
}

func (sd *ServiceDiscovery) Deregister() error {
	sd.registrar.Deregister()
	return nil
}
