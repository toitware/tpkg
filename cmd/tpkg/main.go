// Copyright (C) 2021 Toitware ApS. All rights reserved.

package main

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/toitware/tpkg/commands"
	"github.com/toitware/tpkg/pkg/tpkg"
	"github.com/toitware/tpkg/pkg/tracking"
)

type withExitCode interface {
	ExitCode() int
}

type withSilent interface {
	Silent() bool
}

var (
	// Used for flag.
	cfgFile             string
	cacheDir            string
	noDefaultRegistry   bool
	shouldPrintTracking bool
	sdkVersion          string

	rootCmd = &cobra.Command{
		Use:              "tpkg",
		Short:            "Run pkg commands",
		TraverseChildren: true,
	}
)

func main() {
	cobra.OnInitialize(initConfig)
	// We use the configurations in the viperConf below.
	// If we didn't want to use the globals we could also switch to
	// a PreRun function.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")
	rootCmd.MarkPersistentFlagRequired("config")
	rootCmd.PersistentFlags().StringVar(&cacheDir, "cache", "", "cache dir")
	rootCmd.MarkPersistentFlagRequired("cache")
	rootCmd.PersistentFlags().BoolVar(&noDefaultRegistry, "no-default-registry", false, "Don't use default registry if none exists")
	rootCmd.PersistentFlags().BoolVar(&shouldPrintTracking, "track", false, "Print tracking information")
	rootCmd.PersistentFlags().StringVar(&sdkVersion, "sdk-version", "", "The SDK version")

	runWrapper := func(f commands.CobraErrorCommand) commands.CobraCommand {
		return func(cmd *cobra.Command, args []string) {
			err := f(cmd, args)
			if err != nil {
				_, ok := err.(withSilent)
				if !ok {
					fmt.Fprintf(os.Stderr, "Unhandled error: %v\n", err)
				}
				e, ok := err.(withExitCode)
				if ok {
					os.Exit(e.ExitCode())
				}
				os.Exit(1)
			}
		}
	}

	track := func(ctx context.Context, te *tracking.TrackingEvent) error {
		if shouldPrintTracking {
			tmpl := template.Must(template.New("tracking").Parse(`Category: {{.Category}}
Action: {{.Action}}
Label: {{.Label}}
{{if .Fields }}Fields:{{ range $field, $value := .Fields }}
  {{$field}}: {{$value}}{{end}}{{end}}
`))
			out := bytes.Buffer{}
			if err := tmpl.Execute(&out, te); err != nil {
				log.Fatal("Unexpected error while using template. %w", err)
			}
			fmt.Print(out.String())
		}
		return nil
	}

	pkgCmd, err := commands.Pkg(runWrapper, track, &viperConf{}, nil)
	if err != nil {
		e, ok := err.(withSilent)
		if !ok {
			fmt.Fprintln(os.Stderr, e)
		}
	}
	rootCmd.AddCommand(pkgCmd)
	rootCmd.Execute()
}

func initConfig() {
	viper.SetConfigFile(cfgFile)
	viper.ReadInConfig()
}

type viperConf struct{}

func (t *viperConf) GetPackageCachePaths() ([]string, error) {
	return []string{
		filepath.Join(cacheDir, "tpkg"),
	}, nil
}

func (t *viperConf) GetRegistryCachePaths() ([]string, error) {
	return []string{
		filepath.Join(cacheDir, "tpkg-registries"),
	}, nil
}

const registriesConfigKey = "pkg.registries"

func (t *viperConf) GetRegistryConfigs() (tpkg.RegistryConfigs, error) {
	var registries []tpkg.RegistryConfig
	err := viper.UnmarshalKey(registriesConfigKey, &registries)
	if err != nil {
		return nil, err
	}
	return registries, nil
}

func (t *viperConf) SaveRegistryConfigs(configs tpkg.RegistryConfigs) error {
	viper.Set(registriesConfigKey, configs)
	return viper.WriteConfig()
}

func (t *viperConf) HasRegistryConfigs() bool {
	return noDefaultRegistry || viper.IsSet(registriesConfigKey)
}

func (t *viperConf) SDKVersion() (*version.Version, error) {
	if sdkVersion == "" {
		return nil, nil
	}
	return version.NewVersion(sdkVersion)
}
