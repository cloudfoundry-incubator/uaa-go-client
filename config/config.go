package config

import "time"

const (
	DefaultExpirationBufferInSec = 30
)

type Config struct {
	UaaEndpoint           string `yaml:"uaa_endpoint"`
	ClientName            string `yaml:"client_name"`
	ClientSecret          string `yaml:"client_secret"`
	MaxNumberOfRetries    uint32
	RetryInterval         time.Duration
	ExpirationBufferInSec int64
}
