package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	monitoringsaas "github.com/domainry/domainry-monitoring/saas"
)

func main() {
	address := strings.TrimSpace(os.Getenv("ADDR"))
	if address == "" {
		address = ":8090"
	}
	token := strings.TrimSpace(os.Getenv("DOMAINRY_MONITORING_TOKEN"))
	if token == "" {
		log.Fatal("DOMAINRY_MONITORING_TOKEN is required")
	}
	server := &http.Server{Addr: address, Handler: monitoringsaas.New(monitoringsaas.Options{BearerToken: token}).Routes(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	lifecycle, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-lifecycle.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	log.Printf("Domainry Monitoring listening on %s", address)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
