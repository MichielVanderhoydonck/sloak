package cliutil

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type configKey struct{}

// LoadBaseConfig reads configuration from the given cfgFile (or default .sloak.yaml locations)
// into a fresh, isolated *viper.Viper instance without touching the global viper singleton.
func LoadBaseConfig(cfgFile string) (*viper.Viper, error) {
	v := viper.New()
	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err == nil {
			v.AddConfigPath(home)
			v.AddConfigPath(".")
			v.SetConfigType("yaml")
			v.SetConfigName(".sloak")
		}
	}

	v.SetEnvPrefix("SLOAK")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if cfgFile != "" {
			return nil, fmt.Errorf("reading config file %q: %w", cfgFile, err)
		}
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, fmt.Errorf("reading config file: %w", err)
		}
	}

	return v, nil
}

// WithBaseConfig attaches a base *viper.Viper configuration to the command's context.
func WithBaseConfig(cmd *cobra.Command, v *viper.Viper) {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	cmd.SetContext(context.WithValue(ctx, configKey{}, v))
}

// ScopedConfig returns an isolated *viper.Viper instance initialized with
// any loaded config file settings from the command context, environment variables,
// and the command's flags. This avoids mutating or reading the global viper singleton.
func ScopedConfig(cmd *cobra.Command) (*viper.Viper, error) {
	v := viper.New()
	if ctx := cmd.Context(); ctx != nil {
		if base, ok := ctx.Value(configKey{}).(*viper.Viper); ok && base != nil {
			if settings := base.AllSettings(); len(settings) > 0 {
				if err := v.MergeConfigMap(settings); err != nil {
					return nil, fmt.Errorf("merging config settings: %w", err)
				}
			}
		}
	}
	v.SetEnvPrefix("SLOAK")
	v.AutomaticEnv()
	if err := v.BindPFlags(cmd.Flags()); err != nil {
		return nil, fmt.Errorf("binding flags: %w", err)
	}
	return v, nil
}
