package gonnect

import (
	"errors"
	"io"

	"github.com/spf13/viper"
)

var ErrConfigNoProfileSelected = errors.New("No Profile selected; Set CurrentProfile in the config file or set GONNECT_PROFILE")
var ErrConfigProfileNotFound = errors.New("Profile not found!")

type Config struct {
	CurrentProfile string
	Profiles       map[string]Profile
}

type Profile struct {
	Port          int
	BaseUrl       string
	Store         StoreConfiguration
	SignedInstall bool
}

type StoreConfiguration struct {
	Type        string
	DatabaseUrl string
}

func NewConfig(configFile io.Reader) (*Profile, string, error) {
	LOG.Info("Initializing Configuration")

	runtimeViper := viper.New()
	runtimeViper.SetDefault("CurrentProfile", "dev")
	runtimeViper.BindEnv("CurrentProfile", "GONNECT_PROFILE")
	runtimeViper.BindEnv("DATABASE_URL", "DATABASE_URL")
	runtimeViper.SetConfigType("json")
	config := &Config{}

	err := runtimeViper.ReadConfig(configFile)
	if err != nil {
		return nil, "", err
	}
	err = runtimeViper.Unmarshal(config)
	if err != nil {
		return nil, "", err
	}

	if config.CurrentProfile == "" {
		return nil, "", ErrConfigNoProfileSelected
	}

	if profile, ok := config.Profiles[config.CurrentProfile]; !ok {
		return nil, "", ErrConfigProfileNotFound
	} else {
		runtimeViper.BindEnv("DatabaseUrl", "DATABASE_URL")
		if runtimeViper.IsSet("DatabaseUrl") {
			profile.Store.DatabaseUrl = runtimeViper.GetString("DatabaseUrl")
		}
		return &profile, config.CurrentProfile, nil
	}
}
