// Copyright (C) 2021 Toitware ApS. All rights reserved.

package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/alessio/shellescape"
	"github.com/spf13/cobra"
	"github.com/toitware/toit.git/tools/tpkg/pkg/tpkg"
	"github.com/toitware/toit.git/tools/tpkg/pkg/tracking"
)

type Config interface {
	GetPackageCachePaths() ([]string, error)
	GetRegistryCachePaths() ([]string, error)
	HasRegistryConfigs() bool
	GetRegistryConfigs() (tpkg.RegistryConfigs, error)
	SaveRegistryConfigs(configs tpkg.RegistryConfigs) error
}

var defaultRegistry = tpkg.RegistryConfig{
	Name: "toit",
	Kind: tpkg.RegistryKindGit,
	Path: "github.com/toitware/registry",
}

func (h *pkgHandler) getRegistryConfigsOrDefault() (tpkg.RegistryConfigs, error) {
	if h.cfg.HasRegistryConfigs() {
		return h.cfg.GetRegistryConfigs()
	}
	return []tpkg.RegistryConfig{defaultRegistry}, nil
}

type CobraCommand func(cmd *cobra.Command, args []string)
type CobraErrorCommand func(cmd *cobra.Command, args []string) error
type Run func(CobraErrorCommand) CobraCommand

type Registries tpkg.Registries

func (h *pkgHandler) buildCache() (tpkg.Cache, error) {
	pkgCachePaths, err := h.cfg.GetPackageCachePaths()
	if err != nil {
		return tpkg.Cache{}, err
	}
	registryCachePaths, err := h.cfg.GetRegistryCachePaths()
	if err != nil {
		return tpkg.Cache{}, err
	}
	return tpkg.NewCache(pkgCachePaths, registryCachePaths, h.ui), nil
}

func (h *pkgHandler) buildManager(ctx context.Context) (*tpkg.Manager, error) {
	cache, err := h.buildCache()
	if err != nil {
		return nil, err
	}
	registries, err := h.loadUserRegistries(ctx, cache)
	if err != nil {
		return nil, err
	}
	return tpkg.NewManager(tpkg.Registries(registries), cache, h.ui, h.track), nil
}

func (h *pkgHandler) buildProjectPkgManager(cmd *cobra.Command) (*tpkg.ProjectPkgManager, error) {
	projectRoot, err := cmd.Flags().GetString("project-root")
	if err != nil {
		return nil, err
	}
	manager, err := h.buildManager(cmd.Context())
	if err != nil {
		return nil, err
	}
	paths, err := tpkg.NewProjectPaths(projectRoot, "", "")
	if err != nil {
		return nil, err
	}
	return tpkg.NewProjectPkgManager(manager, paths), nil
}

type pkgHandler struct {
	cfg   Config
	ui    tpkg.UI
	track tracking.Track
}

func Pkg(run Run, track tracking.Track, config Config, ui tpkg.UI) (*cobra.Command, error) {

	// Intercepts any error and checks if it is an already-reported error.
	// If it is, replaces it with a silent error.
	// Otherwise returns it to the caller.
	// Also wraps the call into the given 'run' function.
	errorRun := func(f CobraErrorCommand) CobraCommand {
		return run(func(cmd *cobra.Command, args []string) error {
			err := f(cmd, args)

			if tpkg.IsErrAlreadyReported(err) {
				return newExitError(1)
			}
			return err
		})
	}

	if ui == nil {
		ui = tpkgUI
	}

	handler := &pkgHandler{
		cfg:   config,
		ui:    ui,
		track: track,
	}

	cmd := &cobra.Command{
		Use:   "pkg",
		Short: "Manage packages",
	}
	cmd.PersistentFlags().String("project-root", "", "Specify the project root")

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Creates a new package and lock file in the current directory",
		Long: `Initializes the current directory as the root of the project.

This is done by creating a 'package.lock' and 'package.yaml' file.

If the --project-root flag is used, initializes that directory instead.`,
		Run:  errorRun(handler.pkgInit),
		Args: cobra.NoArgs,
	}
	initCmd.Flags().Bool("pkg", false, "Create a package file")
	initCmd.Flags().Bool("app", false, "Create a lock file for an application")
	cmd.AddCommand(initCmd)

	installCmd := &cobra.Command{
		Use:   "install [<package>]",
		Short: "Installs a package in the current project, or downloads all dependencies",
		Long: `If no 'package' is given, then the command downloads all dependencies.
If necessary, updates the lock-file. This can happen if the lock file doesn't exist
yet, or if the lock-file has local path dependencies (which could have their own
dependencies changed). Recomputation of the dependencies can also be forced by
providing the '--recompute' flag.

If a 'package' is given finds the package with the given name or URL and installs it.
The given 'package' string must uniquely identify a package in the registry.
It is matched against all package names, and URLs. For the names, a package is considered
a match if the string is equal. For URLs it is a match if the string is a complete match, or
the '/' + string is a suffix of the URL.

The 'package' may be suffixed by a version with a '@' separating the package name and
the version. The version doesn't need to be complete. For example 'foo@2' installs
the package foo with the highest version satisfying '2.0.0 <= version < 3.0.0'.
Note: the version constraint in the package.yaml is set to accept semver compatible
versions. If necessary, modify the constraint in that file.

The prefix of the newly installed package is the given prefix, or, if the
'--prefix' argument wasn't provided, the name of the package (if it is a
valid identifier) is used instead.

Once installed, packages can be used by 'import prefix'.

If the '--local' flag is used, then the 'package' argument is interpreted as
a local path to a package directory. Note that published packages may not
contain local packages.
`,
		Example: `  # Ensures all dependencies are downloaded.
  toit pkg install

  # Install package named 'morse'. The prefix is 'morse' (the package name).
  toit pkg install morse

  # Install the package 'morse' with a prefix.
  toit pkg install morse --prefix=prefix_morse

  # Install the version 1.0.0 of the package 'morse'.
  toit pkg install morse@1.0.0

  # Install the package 'morse' by URL (to disambiguate). The longer the URL
  # the less likely a conflict.
  # The prefix is the package name.
  toit pkg install toitware/toit-morse
  toit pkg install github.com/toitware/toit-morse

  # Install the package 'morse' by URL with a prefix.
  toit pkg install toitware/toit-morse --prefix=prefix_morse

  # Install a local package folder by path.
  toit pkg install --local ../my_other_package
  toit pkg install --local submodules/my_other_package --prefix=other
`,
		Run:     errorRun(handler.pkgInstall),
		Args:    cobra.MaximumNArgs(1),
		Aliases: []string{"download", "fetch"},
	}
	installCmd.Flags().Bool("local", false, "Treat package argument as local path")
	installCmd.Flags().Bool("recompute", false, "Recompute dependencies")
	installCmd.Flags().String("prefix", "", "The prefix of the package")
	cmd.AddCommand(installCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "uninstall <prefix>",
		Short: "Uninstalls the package with the given prefix",
		Long: `Uninstalls the package with the given prefix.

Removes the prefix entry from the package files.
The downloaded code is not automatically deleted.
`,
		Run:  errorRun(handler.pkgUninstall),
		Args: cobra.ExactArgs(1),
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "update",
		Short: "Updates all packages to their newest versions",
		Long: `Updates all packages to their newest compatible version.

Uses semantic versioning to find the highest compatible version
of each imported package (and their transitive dependencies).
It then updates all packages to these versions.
`,
		Run:  errorRun(handler.pkgUpdate),
		Args: cobra.NoArgs,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "clean",
		Short: "Removes unnecessary packages",
		// TODO(florian): also add "strip" and "tidy" versions.
		Long: `Removes unnecessary packages.

If a package isn't used anymore removes the downloaded files from the
local package cache.
`,
		Run:  errorRun(handler.pkgClean),
		Args: cobra.NoArgs,
	})

	cmd.AddCommand(&cobra.Command{
		Use:    "lockfile",
		Short:  "Prints the content of the lockfile",
		Run:    errorRun(handler.printLockFile),
		Args:   cobra.NoArgs,
		Hidden: true,
	})

	cmd.AddCommand(&cobra.Command{
		Use:    "packagefile",
		Short:  "Prints the content of package.yaml",
		Run:    errorRun(handler.printPackageFile),
		Args:   cobra.NoArgs,
		Hidden: true,
	})

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Lists all available packages",
		Long: `Lists all packages.

If no argument is given, lists all available packages.
If an argument is given, it must point to a registry path. In that case
only the packages from that registry are shown.`,
		Run:  errorRun(handler.pkgList),
		Args: cobra.MaximumNArgs(1),
	}
	listCmd.Flags().BoolP("verbose", "v", false, "Show more information")
	listCmd.Flags().StringP("output", "o", "list", "Defines the output format (valid: 'list', 'json')")
	cmd.AddCommand(listCmd)

	searchCmd := &cobra.Command{
		Use:   "search <name>",
		Short: "Searches for the given name in all packages",
		Long: `Searches for the given 'name'.

Searches in the name, and description entries, as well as in the URLs of
the packages.`,
		Run:  errorRun(handler.pkgSearch),
		Args: cobra.ExactArgs(1),
	}
	searchCmd.Flags().BoolP("verbose", "v", false, "Show more information")
	cmd.AddCommand(searchCmd)

	registryCmd := &cobra.Command{
		Use:   "registry",
		Short: "Manages registries",
	}
	cmd.AddCommand(registryCmd)

	addRegistryCmd := &cobra.Command{
		Use:   "add <name> <URL>",
		Short: "Adds a registry",
		Long: `Adds a registry.

The 'name' of the registry must not be used yet.

By default the 'URL' is interpreted as Git-URL.
If the '--local' flag is used, then the 'URL' is interpreted as local
path to a folder containing package descriptions.`,
		Example: `  # Add the toit registry.
  toit pkg registry add toit github.com/toitware/registry
`,
		Run:  errorRun(handler.pkgRegistryAdd),
		Args: cobra.ExactArgs(2),
	}
	addRegistryCmd.Flags().Bool("local", false, "Registry is local")
	registryCmd.AddCommand(addRegistryCmd)

	removeRegistryCmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Removes a registry",
		Long: `Removes a registry.

The 'name' of the registry you want to remove.`,
		Example: `  # Remove the toit registry.
  toit pkg registry remove toit
`,
		Run:  errorRun(handler.pkgRegistryRemove),
		Args: cobra.ExactArgs(1),
	}
	registryCmd.AddCommand(removeRegistryCmd)

	syncRegistryCmd := &cobra.Command{
		Use:   "sync",
		Short: "Synchronizes all registries",
		Long: `Synchronizes registries.

If no argument is given, synchronizes all registries.
If an argument is given, it must point to a registry path. In that case
only that registry is synchronized.`,
		Run:  errorRun(handler.pkgRegistrySync),
		Args: cobra.ArbitraryArgs,
	}
	registryCmd.AddCommand(syncRegistryCmd)

	listRegistriesCmd := &cobra.Command{
		Use:   "list",
		Short: "List registries",
		Run:   errorRun(handler.pkgRegistriesList),
		Args:  cobra.NoArgs,
	}

	registryCmd.AddCommand(listRegistriesCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "sync",
		Short: "Synchronizes all registries",
		Long: `Synchronizes all registries.

This is an alias for 'pkg registry sync'`,
		Run:  errorRun(handler.pkgRegistrySync),
		Args: cobra.NoArgs,
	})

	describeCmd := &cobra.Command{
		Use:   "describe [<path_or_url>] [<version>]",
		Short: "Generates a description of the given package",
		Long: `Generates a description of the given package.

If no 'path' is given, defaults to the current working directory.
If one argument is given, then it must be a path to a package.
Otherwise, the first argument is interpreted as the URL to the package, and
the second argument must be a version.

A package description is used when publishing packages. It describes the
package to the outside world. This command extracts a description from
the given path.

If the out directory is specified, generates a description file as used
by registries. The actual description file is generated nested in
directories to make the description path unique.`,
		Run:  errorRun(handler.pkgDescribe),
		Args: cobra.MaximumNArgs(2),
	}
	describeCmd.Flags().BoolP("verbose", "v", false, "Show more information")
	describeCmd.Flags().String("out-dir", "", "Output directory of description files")
	describeCmd.Flags().Bool("allow-local-deps", false, "Allow local dependencies and don't report them")
	describeCmd.Flags().Bool("disallow-local-deps", false, "Always disallow local dependencies and report them as error")
	cmd.AddCommand(describeCmd)

	return cmd, nil
}

type exitError struct {
	code int
}

func (e *exitError) ExitCode() int {
	return e.code
}

func (e *exitError) Silent() bool {
	return true
}

func (e *exitError) Error() string {
	return fmt.Sprintf("ExitError - exit code: %d", e.code)
}

func newExitError(code int) *exitError {
	return &exitError{
		code: code,
	}
}

var tpkgUI = tpkg.FmtUI

func (h pkgHandler) pkgInstall(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	m, err := h.buildProjectPkgManager(cmd)

	if err != nil {
		return err
	}
	projectRoot, err := cmd.Flags().GetString("project-root")
	if err != nil {
		return err
	}
	if projectRoot == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		if cwd != m.Paths.ProjectRootPath {
			// Add the project-root flag, and rebuild the command line.
			args := os.Args
			args = append(args, "--project-root="+m.Paths.ProjectRootPath)
			quoted := []string{}
			for _, arg := range args {
				quoted = append(quoted, shellescape.Quote(arg))
			}
			withFlag := strings.Join(quoted, " ")
			h.ui.ReportError(`Command must be executed in project root.
  Run 'pkg init --app' first to create a new application here, or
  Run with '--project-root': ` + withFlag)
			return newExitError(1)
		}
	}
	isLocal, err := cmd.Flags().GetBool("local")
	if err != nil {
		return err
	}
	prefix, err := cmd.Flags().GetString("prefix")
	if err != nil {
		return err
	}
	forceRecompute, err := cmd.Flags().GetBool("recompute")
	if err != nil {
		return err
	}

	if len(args) == 0 {
		if isLocal {
			h.ui.ReportError("Local flag requires path argument")
			return newExitError(1)
		}
		if prefix != "" {
			h.ui.ReportError("Prefix flag can only be used with package name")
			return newExitError(1)
		}
		err = m.Install(ctx, forceRecompute)

		action := "install-fetch"
		if forceRecompute {
			action = "install-recompute"
		}
		h.track(ctx, &tracking.TrackingEvent{
			Category: "pkg",
			Action:   action,
		})

		if err != nil {
			return err

		}
		return nil
	}

	if forceRecompute {
		h.ui.ReportError("The '--recompute' flag  can only be used without arguments")
	}

	p := args[0]
	installedPrefix, pkgString, err := m.InstallPkg(ctx, isLocal, prefix, p)

	if err != nil {
		return err

	}
	tpkgUI.ReportInfo("Package '%s' installed with prefix '%s'", pkgString, installedPrefix)

	h.track(ctx, &tracking.TrackingEvent{
		Category: "pkg",
		Action:   "install",
		Fields: map[string]string{
			"pkg-string": pkgString,
		},
	})

	return nil
}

func (h pkgHandler) pkgUninstall(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	m, err := h.buildProjectPkgManager(cmd)
	if err != nil {
		return err
	}
	return m.Uninstall(ctx, args[0])

}

func (h pkgHandler) pkgUpdate(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	m, err := h.buildProjectPkgManager(cmd)
	if err != nil {
		return err
	}
	return m.Update(ctx)
}

func (h pkgHandler) pkgClean(cmd *cobra.Command, args []string) error {
	m, err := h.buildProjectPkgManager(cmd)
	if err != nil {
		return err
	}
	return m.CleanPackages()
}

func (h pkgHandler) printLockFile(cmd *cobra.Command, args []string) error {
	m, err := h.buildProjectPkgManager(cmd)
	if err != nil {
		return err
	}
	return m.PrintLockFile()
}

func (h pkgHandler) printPackageFile(cmd *cobra.Command, args []string) error {
	m, err := h.buildProjectPkgManager(cmd)
	if err != nil {
		return err
	}
	return m.PrintSpecFile()
}

func (h pkgHandler) pkgInit(cmd *cobra.Command, args []string) error {
	isPkg, err := cmd.Flags().GetBool("pkg")
	if err != nil {
		return err
	}
	isApp, err := cmd.Flags().GetBool("app")
	if err != nil {
		return err
	}
	if isPkg || isApp {
		h.ui.ReportWarning("The --app and --pkg flags are deprecated")
	}

	projectRoot, err := cmd.Flags().GetString("project-root")
	if err != nil {
		return err
	}

	err = tpkg.InitDirectory(projectRoot, tpkgUI)
	if IsAlreadyExistsError(err) {
		return h.ui.ReportError(ErrorMessage(err))
	} else if err != nil {
		return err

	}
	return nil
}

// Loads all registries as specified by the user's configuration.
func (h *pkgHandler) loadUserRegistries(ctx context.Context, cache tpkg.Cache) ([]tpkg.Registry, error) {
	configs, err := h.getRegistryConfigsOrDefault()
	if err != nil {
		return nil, err
	}
	sync := false
	return configs.Load(ctx, sync, cache, h.ui)
}

func printDesc(d *tpkg.Desc, indent string, isVerbose bool, isJson bool) {
	if isJson {
		md, err := json.Marshal(d)
		if err != nil {
			log.Fatal("Unexpected error marshaling description. %w", err)
		}
		fmt.Println(string(md))
		return
	}
	if !isVerbose {
		fmt.Printf("%s%s - %s\n", indent, d.Name, d.Version)
		return
	}
	tmpl := template.Must(template.New("description").Parse(`{{.Name}}:
  description: {{.Description}}
  url: {{.URL}}
  version: {{.Version}}
  {{if .License}}license: {{.License}}
  {{end}}{{if .Hash}}hash: {{.Hash}}
  {{end}}{{if .Deps }}Dependencies:{{ range $_, $d := .Deps }}
    {{$d.URL}} - {{$d.Version}}{{ end}}{{end}}`))
	out := bytes.Buffer{}
	if err := tmpl.Execute(&out, d); err != nil {
		log.Fatal("Unexpected error while using template. %w", err)
	}
	str := out.String()
	// Add the indentation.
	lines := strings.Split(str, "\n")
	for _, line := range lines {
		fmt.Printf("%s%s\n", indent, line)
	}
}

func (h *pkgHandler) pkgList(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	cache, err := h.buildCache()
	if err != nil {
		return err
	}
	registries, err := h.loadUserRegistries(ctx, cache)
	if err != nil {
		return err
	}
	isVerbose, err := cmd.Flags().GetBool("verbose")
	if err != nil {
		return err
	}
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}
	isJson := output == "json"

	if len(args) == 1 {
		registry := tpkg.NewLocalRegistry("", args[0])
		sync := false
		if err := registry.Load(ctx, sync, cache, h.ui); err != nil {
			if !tpkg.IsErrAlreadyReported(err) {
				return h.ui.ReportError("Error while loading registry '%s': %v", args[0], err)
			}
			return err
		}
		registries = []tpkg.Registry{
			registry,
		}
	}
	for _, registry := range registries {
		fmt.Printf("%s:\n", registry.Describe())
		for _, desc := range registry.Entries() {
			printDesc(desc, "  ", isVerbose, isJson)
		}
	}
	return nil
}

func (h *pkgHandler) pkgRegistriesList(cmd *cobra.Command, args []string) error {
	configs, err := h.getRegistryConfigsOrDefault()
	if err != nil {
		return err
	}
	for _, config := range configs {
		fmt.Printf("%s: %s (%s)\n", config.Name, config.Path, config.Kind)
	}
	return nil
}

func (h *pkgHandler) pkgRegistryAdd(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	cache, err := h.buildCache()
	if err != nil {
		return err
	}
	isLocal, err := cmd.Flags().GetBool("local")
	if err != nil {
		return err
	}
	name := args[0]
	pathOrURL := args[1]
	var kind tpkg.RegistryKind = tpkg.RegistryKindGit
	if isLocal {
		kind = tpkg.RegistryKindLocal
		abs, err := filepath.Abs(pathOrURL)
		if err != nil {
			h.ui.ReportError("Invalid registry: %v", err)
			return newExitError(1)
		}
		info, err := os.Stat(abs)
		if os.IsNotExist(err) {
			h.ui.ReportError("Path doesn't exist: %v", err)
			return newExitError(1)
		} else if !info.IsDir() {
			h.ui.ReportError("Path isn't a directory: %v", err)
			return newExitError(1)
		}
		pathOrURL = abs
	}
	configs, err := h.getRegistryConfigsOrDefault()
	if err != nil {
		return err
	}
	// Check that we don't already have a registry with that name.
	for _, config := range configs {
		if config.Name == name {
			if config.Kind != kind || config.Path != pathOrURL {
				h.ui.ReportError("Registry '%s' already exists", name)
				return newExitError(1)
			}
			// Already exists with the same config.
			if h.cfg.HasRegistryConfigs() {
				return nil
			}
			// Already exists, but not saved in the configuration file.
			// Not strictly necessary, but if the user explicitly adds a configuration
			// we want to write it into the config file.
			return h.cfg.SaveRegistryConfigs(configs)
		}
	}
	registryConfig := tpkg.RegistryConfig{
		Name: name,
		Kind: kind,
		Path: pathOrURL,
	}
	trackingFields := map[string]string{
		"kind": string(kind),
	}
	if kind == tpkg.RegistryKindGit {
		trackingFields["url"] = pathOrURL
	}
	h.track(ctx, &tracking.TrackingEvent{
		Category: "pkg",
		Action:   "registry add",
		Fields:   trackingFields,
	})

	sync := true
	_, err = registryConfig.Load(ctx, sync, cache, h.ui)

	if err != nil {
		if !tpkg.IsErrAlreadyReported(err) {
			return h.ui.ReportError("Registry '%s' has errors: %v", name, err)
		}
		return err
	}
	configs = append(configs, registryConfig)
	return h.cfg.SaveRegistryConfigs(configs)
}

func (h *pkgHandler) pkgRegistryRemove(cmd *cobra.Command, args []string) error {
	name := args[0]
	configs, err := h.cfg.GetRegistryConfigs()
	if err != nil {
		return err
	}
	index := -1
	for i, config := range configs {
		if config.Name == name {
			index = i
			break
		}
	}

	if index == -1 {
		h.ui.ReportError("Registry '%s' does not exist", name)
		return newExitError(1)
	}

	h.track(cmd.Context(), &tracking.TrackingEvent{
		Category: "pkg",
		Action:   "registry remove",
		Fields: map[string]string{
			"path": configs[index].Path,
		},
	})

	configs = append(configs[0:index], configs[index+1:]...)
	return h.cfg.SaveRegistryConfigs(configs)
}

func (h *pkgHandler) pkgRegistrySync(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	cache, err := h.buildCache()
	if err != nil {
		return err
	}
	configs, err := h.getRegistryConfigsOrDefault()
	if err != nil {
		return err
	}

	var configsToSync []tpkg.RegistryConfig

	syncAll := len(args) == 0
	if syncAll {
		configsToSync = configs
	} else {
		nameToConfig := map[string]tpkg.RegistryConfig{}
		for _, config := range configs {
			nameToConfig[config.Name] = config
		}
		for _, toSyncName := range args {
			config, ok := nameToConfig[toSyncName]
			if !ok {
				h.ui.ReportWarning("Config '%s' not found", toSyncName)
			} else {
				configsToSync = append(configsToSync, config)
			}
		}
	}

	hasErrors := false
	for _, config := range configsToSync {
		sync := true
		h.ui.ReportInfo("Syncing '%s'", config.Name)
		_, err := config.Load(ctx, sync, cache, h.ui)
		if err != nil {
			if !tpkg.IsErrAlreadyReported(err) {
				h.ui.ReportError("Error while syncing '%s': '%v'", config.Name, err)
			} else {
				h.ui.ReportError("Error while syncing '%s'", config.Name)
			}
			hasErrors = true
		}
	}
	if hasErrors {
		return newExitError(1)
	}
	return nil
}

func (h *pkgHandler) pkgSearch(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	h.track(ctx, &tracking.TrackingEvent{
		Category: "pkg",
		Action:   "search",
		Fields: map[string]string{
			"needle": args[0],
		},
	})

	cache, err := h.buildCache()
	if err != nil {
		return err
	}
	registries, err := h.loadUserRegistries(ctx, cache)
	if err != nil {
		return err
	}
	isVerbose, err := cmd.Flags().GetBool("verbose")
	if err != nil {
		return err
	}

	found, err := tpkg.Registries(registries).SearchAll(args[0])
	if err != nil {
		return err
	}
	found, err = found.WithoutLowerVersions(nil)
	if err != nil {
		return err
	}
	for _, descReg := range found {
		printDesc(descReg.Desc, "", isVerbose, false)
	}
	return nil
}

func (h *pkgHandler) pkgDescribe(cmd *cobra.Command, args []string) error {
	isVerbose, err := cmd.Flags().GetBool("verbose")
	if err != nil {
		return err
	}
	outDir, err := cmd.Flags().GetString("out-dir")
	if err != nil {
		return err
	}
	var desc *tpkg.Desc
	if len(args) < 2 && outDir != "" {
		h.ui.ReportError("The --out-dir flag requires a URL and version")
		return newExitError(1)
	}

	allowFlag, err := cmd.Flags().GetBool("allow-local-deps")
	if err != nil {
		return err
	}
	disallowFlag, err := cmd.Flags().GetBool("disallow-local-deps")
	if err != nil {
		return err
	}

	if allowFlag && disallowFlag {
		h.ui.ReportError("--allow-local-deps and --disallow-local-deps are exclusive")
		return newExitError(1)
	}

	var allowsLocalDeps = tpkg.ReportLocalDeps
	if allowFlag {
		allowsLocalDeps = tpkg.AllowLocalDeps
	} else if disallowFlag || len(args) >= 2 {
		allowsLocalDeps = tpkg.DisallowLocalDeps
	}

	if len(args) == 0 {
		var cwd string
		cwd, err = os.Getwd()
		if err != nil {
			return err
		}

		desc, err = tpkg.ScrapeDescriptionAt(cwd, allowsLocalDeps, isVerbose, h.ui)
	} else if len(args) == 1 {
		desc, err = tpkg.ScrapeDescriptionAt(args[0], allowsLocalDeps, isVerbose, h.ui)
	} else {
		h.track(cmd.Context(), &tracking.TrackingEvent{
			Category: "pkg",
			Action:   "describe",
			Fields: map[string]string{
				"url":     args[0],
				"version": args[1],
			},
		})

		ctx := cmd.Context()
		desc, err = tpkg.ScrapeDescriptionGit(ctx, args[0], args[1], allowsLocalDeps, isVerbose, h.ui)
	}

	if err != nil {
		return err
	}
	if outDir == "" {
		printDesc(desc, "", true, false)
		return nil
	}
	descPath, err := desc.WriteInDir(outDir)
	if err != nil {
		return err
	}
	h.ui.ReportInfo("Wrote '%s'", descPath)
	return nil
}
