package main

import (
	"crypto/x509"
	"device-manufacturing-system/pkg/manufacturing/api"
	"device-manufacturing-system/pkg/manufacturing/client/scep"
	"device-manufacturing-system/pkg/manufacturing/configs"
	"device-manufacturing-system/pkg/manufacturing/ocsp"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
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

	client := scep.NewClient(cfg.CertFile, cfg.KeyFile, cfg.ProxyAddress, cfg.SCEPMapping)

	var s api.Service
	{
		s = api.NewDeviceService(cfg.CAPath, cfg.AuthKeyFile, client)
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

func verifyPeerCertificateParallel(_ [][]byte, verifiedChains [][]*x509.Certificate) (err error) {
	for i := 0; i < len(verifiedChains); i++ {
		var wg sync.WaitGroup
		n := len(verifiedChains[i]) - 1
		wg.Add(n)
		ocspStatusChan := make(chan ocsp.OCSPStatus, n)
		ocspErrorChan := make(chan error, n)
		for j := 0; j < n; j++ {
			go ocsp.GetRevocationStatus(&wg, ocspStatusChan, ocspErrorChan, verifiedChains[i][j], verifiedChains[i][j+1])
		}
		results := make([]ocsp.OCSPStatus, n)
		for j := 0; j < n; j++ {
			results[j] = <-ocspStatusChan
		}
		close(ocspErrorChan)
		close(ocspStatusChan)
		wg.Wait()
		for _, r := range results {
			if r != ocsp.OCSPSuccess {
				e := <-ocspErrorChan
				return fmt.Errorf("failed certificate revocation check. err: %v", e)
			}
		}
	}
	return nil
}
