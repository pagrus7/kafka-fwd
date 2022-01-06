package config

import (
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"time"
)

var (
	ErrNoRoutes      = errors.New("no routes defined")
	ErrDefIncomplete = errors.New("route definition incomplete")

	RouteDefaults = RouteDef{
		PauseThreshold:  100,
		ResumeThreshold: 10,
	}
)

type RouteConfig struct {
	RouteDefs map[string]*RouteDef `mapstructure:"forward"`
}

type RouteDef struct {
	From            string        // derived from map key
	To              string        `mapstructure:"to"`
	Delay           time.Duration `mapstructure:"delay"`
	PauseThreshold  int           `mapstructure:"pause-threshold"`
	ResumeThreshold int           `mapstructure:"resume-threshold"`
}

func ProvideRouteConfig(viper *viper.Viper) (*RouteConfig, error) {
	appConf := RouteConfig{}
	if err := viper.Unmarshal(&appConf); err != nil {
		return nil, err
	}

	if err := appConf.complete(); err != nil {
		return nil, err
	}
	return &appConf, nil
}

func (ac *RouteConfig) complete() error {
	if len(ac.RouteDefs) == 0 {
		return ErrNoRoutes
	}
	for name, def := range ac.RouteDefs {
		def.From = name

		if def == nil || def.From == "" || def.To == "" {
			return fmt.Errorf("error in route '%s': %w", name, ErrDefIncomplete)
		}

		if def.ResumeThreshold <= 0 {
			def.ResumeThreshold = RouteDefaults.ResumeThreshold
		}

		if def.PauseThreshold <= 0 {
			def.PauseThreshold = RouteDefaults.PauseThreshold
		}

		if def.PauseThreshold <= def.ResumeThreshold {
			def.ResumeThreshold = RouteDefaults.ResumeThreshold
			def.PauseThreshold = RouteDefaults.PauseThreshold
		}
	}
	return nil
}
