package config

import "os"

type Config struct {
	DAVURL   string
	Username string
	Password string
}

func Load() Config {

	return Config{
		DAVURL:   os.Getenv("DAV_URL"),
		Username: os.Getenv("DAV_USERNAME"),
		Password: os.Getenv("DAV_PASSWORD"),
	}
}
