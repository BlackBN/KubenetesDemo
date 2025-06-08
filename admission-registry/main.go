package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/BlackBN/KubenetesDemo/admission-registry/pkg/types"
	klog "k8s.io/klog/v2"
)

func main() {
	//是需要tls的
	var params types.WebhookServerParams
	flag.Int64Var(&params.Port, "port", 443, "webhook listen port")
	flag.StringVar(&params.CertFile, "tlsCertFile", "/etc/webhook/certs/tls.crt", "x509 certification file")
	flag.StringVar(&params.KeyFile, "tlsKeyFile", "/etc/webhook/certs/tls.key", "x509 private key file")
	flag.Parse()
	certficate, err := tls.LoadX509KeyPair(params.CertFile, params.KeyFile)
	if err != nil {
		klog.Errorf("Failed to load key pair: %v", err)
		panic(err)
	}

	webhookServer := &types.WebhookServer{
		Server: &http.Server{
			Addr: fmt.Sprintf(":%d", params.Port),
			TLSConfig: &tls.Config{
				Certificates: []tls.Certificate{
					certficate,
				},
			},
		},
		WhiteListRegistries: strings.Split(os.Getenv("WHITELIST_REGISTRIES"), ","),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/validate", webhookServer.Handler)
	mux.HandleFunc("/mutate", webhookServer.Handler)
	webhookServer.Server.Handler = mux
	go func() {
		if err := webhookServer.Server.ListenAndServeTLS("", ""); err != nil {
			klog.Errorf("failed to listen webhook server: %v", err)
		}
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	klog.Infof("get os shtdown signal")

	if err := webhookServer.Server.Shutdown(context.Background()); err != nil {
		klog.Errorf("failed to shutdown webhookserver: %v", err)
	}

}
