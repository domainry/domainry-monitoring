package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/domainry/domainry-foundation/logging"
	"github.com/domainry/domainry-foundation/processhost"
	monitoringsaas "github.com/domainry/domainry-monitoring/saas"
)

func main() {
	if run() != nil {
		os.Exit(1)
	}
}

func run() (runErr error) {
	lifecycle, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	observability, err := processhost.Open(lifecycle, processhost.Config{ServiceName: "domainry-monitoring"})
	if err != nil {
		return err
	}
	started := false
	startupFailure := logging.Failure{Stage: "configuration_validate"}
	defer func() {
		if !started && runErr != nil {
			observability.StartFailed(context.Background(), runErr, startupFailure)
			runErr = errors.Join(runErr, observability.Close(context.Background(), nil))
			return
		}
		runErr = observability.Finish(context.Background(), nil, runErr)
	}()
	address := strings.TrimSpace(os.Getenv("ADDR"))
	if address == "" {
		address = ":8090"
	}
	token := strings.TrimSpace(os.Getenv("DOMAINRY_MONITORING_TOKEN"))
	if token == "" {
		return errors.New("DOMAINRY_MONITORING_TOKEN is required")
	}
	startupFailure = logging.Failure{Stage: "module_assembly"}
	handler, err := monitoringsaas.New(monitoringsaas.Options{BearerToken: token})
	if err != nil {
		return err
	}
	server := &http.Server{Addr: address, Handler: observability.Handler("public", handler.Routes()), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	startupFailure = logging.Failure{Stage: "listener_bind", Dependency: "public_listener"}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(listener) }()
	started = true
	observability.Started(lifecycle, map[string]any{"listener_group": "public"})
	select {
	case <-lifecycle.Done():
		observability.Stopping(context.Background(), map[string]any{"reason": "signal"})
		shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
