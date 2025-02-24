package util

import (
	"github.com/spf13/viper"
)

// Config stores all configuration of the application.
// The values are read by viper from a config file or environment variable.
type Config struct {
	Environment       string `mapstructure:"ENVIRONMENT"`
	HTTPServerAddress string `mapstructure:"HTTP_SERVER_ADDRESS"`
	GRPCServerAddress string `mapstructure:"GRPC_SERVER_ADDRESS"`
	ScyllaKeyspace    string `mapstructure:"SCYLLA_KEYSPACE"`
	ScyllaHosts       string `mapstructure:"SCYLLA_HOSTS"`
	ScyllaPort        string `mapstructure:"SCYLLA_PORT"`
	RedisAddress      string `mapstructure:"REDIS_ADDRESS"`
	RedisPassword     string `mapstructure:"REDIS_PASSWORD"`
	RedisDatabase     string `mapstructure:"REDIS_DATABASE"`
}

// LoadConfig reads configuration from file or environment variables.
func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)
	return
}
