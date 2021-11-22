package config

import (
	"os"
	"path/filepath"
)

func UserConfigPath() (string, error) {
	if path, ok := os.LookupEnv(UserConfigDirEnv); ok {
		return path, nil
	}

	homedir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homedir, ".config", "toit"), nil
}

// UserConfigFile returns the config file in the user directory.
func UserConfigFile() (string, bool) {
	if homedir, err := EnsureDirectory(UserConfigPath()); err == nil {
		return filepath.Join(homedir, "config.yaml"), true
	}
	return "", false
}
