package config

import (
	"fmt"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	*koanf.Koanf
}

func NewConfig(path string) *Config {
	k := koanf.New(".")
	err := k.Load(file.Provider(path), yaml.Parser())
	if err != nil {
		panic(err)
	}
	return &Config{
		Koanf: k,
	}
}

func (cfg *Config) GetPublicHttpPort() string {
	return cfg.String("port.http.public.port")
}

func (cfg *Config) GetPublicHttpPortTimeout() time.Duration {
	return cfg.Duration("port.http.public.timeout")
}

func (cfg *Config) GetStorageType() string {
	return cfg.String("storage.type")
}

func (cfg *Config) GetConnString() string {
	return cfg.String(fmt.Sprintf("%s.%s", cfg.GetStorageType(), "connection_string"))
}

func (cfg *Config) GetClientTimeout() time.Duration {
	return cfg.Duration("client.timeout")
}

func (cfg *Config) GetActualizeInterval() time.Duration {
	return cfg.Duration("cron.actualize.interval")
}

func (cfg *Config) GetActualizeIntervalContextTimeout() time.Duration {
	return cfg.Duration("cron.actualize.timeout")
}

func (cfg *Config) TracerEndpoint() string {
	return cfg.String("tracing.jaeger")
}

func (cfg *Config) IsTracerSwitched() bool {
	return cfg.Bool("tracing.switch_on")
}

func (cfg *Config) GetServiceName() string {
	return cfg.String("service_name")
}

func (cfg *Config) GetServiceVersion() string {
	return cfg.String("version")
}
