package main

import (
	"gopkg.in/yaml.v3"
	_ "gopkg.in/yaml.v3"
	"os"
)

type InfluxDBConfig struct {
	Endpoint string `yaml:"endpoint"`
	Org      string `yaml:"org"`
	Bucket   string `yaml:"bucket"`
	Token    string `yaml:"token"`
}

type VMConfig struct {
	Endpoint string `yaml:"endpoint"`
}

type Config struct {
	InfluxDB *InfluxDBConfig `yaml:"influxdb"`
	VM       *VMConfig       `yaml:"vm"`
}

func InitConfig(cfgPath string) (*Config, error) {
	bs, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, err
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(bs, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
