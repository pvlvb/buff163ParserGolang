package configManager

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type Config struct {
	Mode string `yaml:"mode"`
	//BackendAPIKeyEnv string `yaml:"backend_apikey_env"`
}

func LoadConfig(path string) (*Config, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(bytes, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
