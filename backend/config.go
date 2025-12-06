package cashtrack

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	ServerConfig `envPrefix:"SERVER_" envDefault:""`
}

func loadOptional(file string) error {
	err := godotenv.Load(file) // The Original .env
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("error loading .env file: %w", err)
		}
	} else {
		log.Info().Msgf("Loading environment variables from %s", file)
	}
	return nil
}

func ProvideConfig() (Config, error) {
	appEnv := os.Getenv("CASHTRACK_ENV")
	if "" == appEnv {
		appEnv = "development"
	}
	log.Info().Msgf("Environment: %s", appEnv)

	if err := loadOptional(".env." + appEnv); err != nil {
		return Config{}, err
	}
	if "test" != appEnv {
		if err := loadOptional(".env.local"); err != nil {
			return Config{}, err
		}
	}
	if err := loadOptional(".env"); err != nil {
		return Config{}, err
	}

	var config Config
	err := env.ParseWithOptions(&config, env.Options{RequiredIfNoDef: true, UseFieldNameByDefault: true})
	if err != nil {
		return config, err
	}
	return config, nil
}
