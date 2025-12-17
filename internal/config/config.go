package config

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/vrischmann/envconfig"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

const (
	notificationDelay = 1 * time.Second
)

type Config struct {
	LogLevel  string `envconfig:"default=info" yaml:"logLevel"`
	LogFormat string `envconfig:"default=json" yaml:"logFormat"`
}

type CallbackFn func(Config)

func GetConfig(prefix string) (Config, error) {
	cfg := Config{}
	err := envconfig.InitWithPrefix(&cfg, prefix)
	return cfg, err
}

func LoadLogConfig(path string) (Config, error) {
	cfg := Config{}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	err = yaml.Unmarshal(data, &cfg)
	return cfg, err
}

// RunOnConfigChange - run callback functions when config is changed
func RunOnConfigChange(ctx context.Context, log *zap.SugaredLogger, path string, callbacks ...CallbackFn) {
	log.Info("config notifier started")

	for {
		// wait 1 sec not to burn out the container for example when any method below always ends with an error
		time.Sleep(notificationDelay)

		err := fireCallbacksOnConfigChange(ctx, log, path, callbacks...)
		if err != nil && errors.Is(err, context.Canceled) {
			log.Info("context canceled")
			return
		}
		if err != nil {
			log.Error(err)
		}
	}
}

func fireCallbacksOnConfigChange(ctx context.Context, log *zap.SugaredLogger, path string, callbacks ...CallbackFn) error {
	err := notifyModification(ctx, path)
	if err != nil {
		return err
	}

	log.Info("config file change detected")

	cfg, err := LoadLogConfig(path)
	if err != nil {
		return err
	}

	log.Debugf("firing '%d' callbacks", len(callbacks))

	fireCallbacks(cfg, callbacks...)
	return nil
}

func fireCallbacks(cfg Config, funcs ...CallbackFn) {
	for i := range funcs {
		fn := funcs[i]
		fn(cfg)
	}
}

// notifyModification watches for file modifications using fsnotify
func notifyModification(ctx context.Context, path string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer func() {
		_ = watcher.Close()
	}()

	// Watch the directory containing the config file to catch Kubernetes ConfigMap updates
	// which are done via atomic symlink changes
	configDir := filepath.Dir(path)
	if err := watcher.Add(configDir); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event := <-watcher.Events:
			// Kubernetes ConfigMap updates trigger Create events on the ..data symlink
			if event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Write == fsnotify.Write {
				return nil
			}
		case err := <-watcher.Errors:
			return err
		}
	}
}
