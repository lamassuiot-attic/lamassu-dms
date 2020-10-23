package main

import (
	"device-manufacturing-system/pkg/api"
	"device-manufacturing-system/pkg/configs"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-kit/kit/log"
)

func main() {

	cfg, err := configs.NewConfig()
	if err != nil {
		panic(err)
	}

	var (
		httpAddr = flag.String("http.addr", ":"+cfg.Port, "HTTP listen address")
	)
	flag.Parse()

	var logger log.Logger
	{
		logger = log.NewLogfmtLogger(os.Stderr)
		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
		logger = log.With(logger, "caller", log.DefaultCaller)
	}

	var s api.Service
	{
		s = api.NewDeviceService(cfg.CAPath, cfg.CertFile, cfg.KeyFile, cfg.SCEPMapping)
		s = api.LoggingMidleware(logger)(s)
	}

	mux := http.NewServeMux()

	mux.Handle("/v1/", api.MakeHTTPHandler(s, log.With(logger, "component", "HTTP")))
	http.Handle("/", accessControl(mux, cfg.UIProtocol, cfg.UIHost, cfg.UIPort))

	errs := make(chan error)
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errs <- fmt.Errorf("%s", <-c)
	}()

	go func() {
		logger.Log("transport", "HTTP", "addr", "httpAddr")
		errs <- http.ListenAndServeTLS(*httpAddr, cfg.CertFile, cfg.KeyFile, nil)
	}()

	logger.Log("exit", <-errs)

}

func accessControl(h http.Handler, UIProtocol string, UIHost string, UIPort string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", UIProtocol+"://"+UIHost+":"+UIPort)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			return
		}

		h.ServeHTTP(w, r)
	})
}
