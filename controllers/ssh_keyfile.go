// Copyright (C) 2024 Toitware ApS. All rights reserved.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package controllers

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/toitware/tpkg/config"
)

func populateSSHKeyFile(config *config.Config) error {
	if config.Registry.SSHKey == "" {
		return nil
	}

	if _, err := os.Stat(config.Registry.SSHKeyFile); os.IsNotExist(err) {
		// Write the content of the config.Registry.SSHKey into the file.
		dir := filepath.Dir(config.Registry.SSHKeyFile)
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("Failed to create directory: '%s'", dir)
		}
		if err := ioutil.WriteFile(config.Registry.SSHKeyFile, []byte(config.Registry.SSHKey), 0600); err != nil {
			return fmt.Errorf("Failed to write SSH key to path: '%s'", config.Registry.SSHKeyFile)
		}
	}
	return nil
}
