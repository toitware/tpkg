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
	"strings"

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
	noAutosync          bool

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
	rootCmd.PersistentFlags().BoolVar(&noAutosync, "no-autosync", false, "Don't automatically sync")
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

	cfgStore := &viperConfigStore{}

	pkgCmd, err := commands.Pkg(runWrapper, track, cfgStore, nil)
	if err != nil {
		e, ok := err.(withSilent)
		if !ok {
			fmt.Fprintln(os.Stderr, e)
		}
	}
	rootCmd.AddCommand(pkgCmd)

	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Get and set package management configuration options",
	}
	autosyncCmd := &cobra.Command{
		Use:   "autosync",
		Short: "Returns or sets the autosync option",
		Long: `Returns or sets the autosync option.

Without argument prints the current value of the option.

If an argument ('true' or 'false') is provided, updates the
option in the configuration.
`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return pkgConfigAutosync(cmd, args, cfgStore)
		},
		Aliases: []string{"auto-sync"},
	}
	configCmd.AddCommand(autosyncCmd)
	rootCmd.AddCommand(configCmd)

	rootCmd.Execute()
}

func initConfig() {
	viper.SetConfigFile(cfgFile)
	viper.ReadInConfig()
}

type viperConfigStore struct{}

const packageInstallPathConfigEnv = "TOIT_PACKAGE_INSTALL_PATH"
const configKeyRegistries = "pkg.registries"
const configKeyAutosync = "pkg.autosync"

func (vc *viperConfigStore) Load(ctx context.Context) (*commands.Config, error) {
	result := commands.Config{}
	result.PackageCachePaths = []string{filepath.Join(cacheDir, "tpkg")}
	result.RegistryCachePaths = []string{filepath.Join(cacheDir, "tpkg-registries")}
	if p, ok := os.LookupEnv(packageInstallPathConfigEnv); ok {
		result.PackageInstallPath = &p
	}
	if sdkVersion != "" {
		v, err := version.NewVersion(sdkVersion)
		if err != nil {
			return nil, err
		}
		result.SDKVersion = v
	}

	var configs tpkg.RegistryConfigs
	if viper.IsSet(configKeyRegistries) {
		err := viper.UnmarshalKey(configKeyRegistries, &configs)
		if err != nil {
			return nil, err
		}
		if configs == nil {
			// Viper seems to just ignore empty lists.
			configs = tpkg.RegistryConfigs{}
		}
	} else if noDefaultRegistry {
		configs = tpkg.RegistryConfigs{}
	}
	result.RegistryConfigs = configs

	var autosync *bool
	if noAutosync {
		sync := false
		autosync = &sync
	} else if viper.IsSet(configKeyAutosync) {
		sync := viper.GetBool(configKeyAutosync)
		autosync = &sync
	}
	result.Autosync = autosync

	return &result, nil
}

func (vc *viperConfigStore) Store(ctx context.Context, cfg *commands.Config) error {
	if cfg.Autosync != nil {
		viper.Set(configKeyAutosync, *cfg.Autosync)
	}
	if cfg.RegistryConfigs != nil {
		viper.Set(configKeyRegistries, cfg.RegistryConfigs)
	}
	return viper.WriteConfig()
}

func pkgConfigAutosync(cmd *cobra.Command, args []string, cfgStore commands.ConfigStore) error {
	ctx := cmd.Context()
	conf, err := cfgStore.Load(ctx)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		if conf.Autosync == nil {
			fmt.Println(true)
		} else {
			fmt.Println(*conf.Autosync)
		}
		return nil
	}
	newValStr := strings.ToLower(args[0])
	if newValStr != "true" && newValStr != "false" {
		msg := fmt.Sprintf("Not a boolean value '%s'", newValStr)
		cobra.CheckErr(msg)
	}
	newVal := newValStr == "true"
	conf.Autosync = &newVal
	return cfgStore.Store(ctx, conf)
}
