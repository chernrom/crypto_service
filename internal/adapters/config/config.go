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
