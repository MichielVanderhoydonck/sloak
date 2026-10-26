package cliutil

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ScopedConfig returns an isolated *viper.Viper instance initialized with
// any loaded config file settings, environment variables, and the command's flags.
// This avoids binding flags directly to the global viper singleton.
func ScopedConfig(cmd *cobra.Command) (*viper.Viper, error) {
	v := viper.New()
	if globalSettings := viper.AllSettings(); len(globalSettings) > 0 {
		if err := v.MergeConfigMap(globalSettings); err != nil {
			return nil, fmt.Errorf("merging config settings: %w", err)
		}
	}
	v.SetEnvPrefix("SLOAK")
	v.AutomaticEnv()
	if err := v.BindPFlags(cmd.Flags()); err != nil {
		return nil, fmt.Errorf("binding flags: %w", err)
	}
	return v, nil
}
