package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"example.com/m/internal/html"
	"example.com/m/internal/http"
	"example.com/m/internal/kubernetes"
)

func New() (App, error) {
	return newWithGetter(os.Getenv)
}

func newWithGetter(get envGetter) (App, error) {
	fmt.Println("Load environment variables")
	cfg, err := loadConfig(get)
	if err != nil {
		return nil, err
	}
	return &app{
		cfg: cfg,
	}, nil
}

func (a *app) Run() error {
	fmt.Println("Run startup checks")
	var errs []error

	k, err := kubernetes.New(a.cfg.KubeconfigPath)
	if err != nil {
		errs = append(errs, err)
	}

	t, err := html.New()
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	fmt.Println("Completed startup checks")

	httpServer := http.New(t, k, a.cfg.Port)

	ctxInterrupt, stopInterrupt := SignalHandling()
	ctxHttpServer, stopHttpServer := context.WithCancel(context.Background())

	go httpServer.Run(stopHttpServer)

	select {
	case <-ctxInterrupt.Done():
		fmt.Println("Interrupt signal received. Close services within 30 seconds.")
		stopInterrupt()
		ctxTimeout, stopTimeout := context.WithTimeout(context.Background(), 30*time.Second)
		defer stopTimeout()

		err := httpServer.Stop(ctxTimeout)
		if err != nil {
			fmt.Println("Error closing http server within timeout")
		}

		return err

	case <-ctxHttpServer.Done():
		fmt.Println("HTTP Server had an error. Close everything")
		stopInterrupt()

	}

	return nil

}

type App interface {
	Run() error
}

type app struct {
	cfg config
}

type config struct {
	Port           int
	KubeconfigPath string
}

type envGetter func(string) string

func loadConfig(get envGetter) (config, error) {
	cfg := config{
		Port:           8080,
		KubeconfigPath: "~/.kube/config",
	}
	var errs []error

	if portStr := get("PORT"); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			errs = append(errs, fmt.Errorf("PORT must be a valid integer"))
		} else if port < 1 || port > 65535 {
			errs = append(errs, fmt.Errorf("PORT must be between 1 and 65535"))
		} else {
			cfg.Port = port
		}
	}

	if kubeconfigPath := get("KUBECONFIG"); kubeconfigPath != "" {
		if !pathExists(kubeconfigPath) {
			errs = append(errs, fmt.Errorf("Kubeconfig not found at %s", kubeconfigPath))
		} else {
			cfg.KubeconfigPath = kubeconfigPath
		}
	}
	cfg.KubeconfigPath = expandPath(cfg.KubeconfigPath)

	if len(errs) > 0 {
		return config{}, errors.Join(errs...)
	}

	return cfg, nil
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

func SignalHandling() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}
