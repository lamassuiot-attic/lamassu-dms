package main

import (
	"device-manufacturing-system/pkg/enroller/auth"
	"device-manufacturing-system/pkg/manufacturing/api"
	"device-manufacturing-system/pkg/manufacturing/client/scep"
	"device-manufacturing-system/pkg/manufacturing/configs"
	"device-manufacturing-system/pkg/manufacturing/discovery/consul"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	jaegercfg "github.com/uber/jaeger-client-go/config"
)

func main() {
	var logger log.Logger
	{
		logger = log.NewJSONLogger(os.Stdout)
		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
		logger = log.With(logger, "caller", log.DefaultCaller)
		logger = level.NewFilter(logger, level.AllowInfo())
	}

	cfg, err := configs.NewConfig("manufacturing")
	if err != nil {
		level.Error(logger).Log("err", err, "msg", "Could not read environment configuration values")
		os.Exit(1)
	}
	level.Info(logger).Log("msg", "Environment configuration values loaded")

	auth := auth.NewAuth(cfg.KeycloakHostname, cfg.KeycloakPort, cfg.KeycloakProtocol, cfg.KeycloakRealm, cfg.KeycloakCA)
	level.Info(logger).Log("msg", "Connection established with authentication system")

	jcfg, err := jaegercfg.FromEnv()
	if err != nil {
		level.Error(logger).Log("err", err, "msg", "Could not load Jaeger configuration values fron environment")
		os.Exit(1)
	}
	level.Info(logger).Log("msg", "Jaeger configuration values loaded")
	tracer, closer, err := jcfg.NewTracer()
	if err != nil {
		level.Error(logger).Log("err", err, "msg", "Could not start Jaeger tracer")
		os.Exit(1)
	}
	defer closer.Close()
	level.Info(logger).Log("msg", "Jaeger tracer started")

	client := scep.NewClient(cfg.CertFile, cfg.KeyFile, cfg.ProxyAddress, cfg.ConsulProtocol, cfg.ConsulHost, cfg.ConsulPort, cfg.ConsulCA, cfg.SCEPMapping, cfg.ProxyCA, logger, tracer)
	level.Info(logger).Log("msg", "Remote SCEP Client started")

	fieldKeys := []string{"method"}
	var s api.Service
	{
		s = api.NewDeviceService(cfg.AuthKeyFile, client)
		s = api.LoggingMidleware(logger)(s)
		s = api.NewInstumentingMiddleware(
			kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
				Namespace: "device_manufacturing_system",
				Subsystem: "manufacturing_service",
				Name:      "request_count",
				Help:      "Number of requests received.",
			}, fieldKeys),
			kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
				Namespace: "device_manufacturing_system",
				Subsystem: "manufacturing_service",
				Name:      "request_latency_microseconds",
				Help:      "Total duration of requests in microseconds.",
			}, fieldKeys),
		)(s)
	}

	consulsd, err := consul.NewServiceDiscovery(cfg.ConsulProtocol, cfg.ConsulHost, cfg.ConsulPort, cfg.ConsulCA, logger)
	if err != nil {
		level.Error(logger).Log("err", err, "msg", "Could not start connection with Consul Service Discovery")
		os.Exit(1)
	}
	err = consulsd.Register("https", "manufacturing", cfg.Port)
	if err != nil {
		level.Error(logger).Log("err", err, "msg", "Could not register service liveness information to Consul")
		os.Exit(1)
	}
	level.Info(logger).Log("msg", "Service liveness information registered to Consul")
	mux := http.NewServeMux()

	mux.Handle("/v1/", api.MakeHTTPHandler(s, log.With(logger, "component", "HTTP"), auth, tracer))
	http.Handle("/", accessControl(mux, cfg.UIProtocol, cfg.UIHost, cfg.UIPort))
	http.Handle("/metrics", promhttp.Handler())

	errs := make(chan error)
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errs <- fmt.Errorf("%s", <-c)
	}()

	go func() {
		level.Info(logger).Log("transport", "HTTPS", "address", ":"+cfg.Port, "msg", "listening")
		errs <- http.ListenAndServeTLS(":"+cfg.Port, cfg.CertFile, cfg.KeyFile, nil)
	}()

	level.Info(logger).Log("exit", <-errs)
	err = consulsd.Deregister()
	if err != nil {
		level.Error(logger).Log("err", err, "msg", "Could not deregister service liveness information from Consul")
		os.Exit(1)
	}
	level.Info(logger).Log("msg", "Service liveness information deregistered from Consul")
}

func accessControl(h http.Handler, UIProtocol string, UIHost string, UIPort string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var uiURL string
		if UIPort == "" {
			uiURL = UIProtocol + "://" + UIHost
		} else {
			uiURL = UIProtocol + "://" + UIHost + ":" + UIPort
		}
		w.Header().Set("Access-Control-Allow-Origin", uiURL)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			return
		}

		h.ServeHTTP(w, r)
	})
}
