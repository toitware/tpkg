// Copyright (C) 2023 Toitware ApS. All rights reserved.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"net/http"
	"time"

	"github.com/toitware/tpkg/config"
	"github.com/toitware/tpkg/controllers"
	"github.com/toitware/tpkg/handlers"
	"github.com/toitware/tpkg/pkg/network"
	"github.com/toitware/tpkg/pkg/service"
	"github.com/toitware/tpkg/pkg/toitdoc"
	"go.uber.org/fx"
)

func main() {
	app := fx.New(
		config.Module,
		handlers.Module,
		service.Module,
		network.Module,
		controllers.Module,
		toitdoc.Module,
	)
	startCtx, _ := context.WithTimeout(context.Background(), app.StartTimeout())
	app.Start(startCtx)
	go func() {
		// Make sure the rest is up and running.
		time.Sleep(10 * time.Second)
		// Connect to our own server (without a timeout) and see if we are running.
		client := http.Client{
			Timeout: 9999 * time.Second,
		}
		url := "http://localhost:8733/github.com/toitware/toit-font-google-100dpi-roboto@1.2.0/docs/roboto/library-summary"
		println("Contacting", url)
		response, err := client.Get(url)
		if err != nil || response.StatusCode != 200 {
			println("************************* TEST FAILED - Failed to load toitdoc ******************", err)
			if response != nil {
				println("Status", response.Status)
			}
		} else {
			println("*************************** TEST SUCCESS - Loaded toitdoc ************************", response.Status)
		}
		stopCtx, _ := context.WithTimeout(context.Background(), app.StopTimeout())
		app.Stop(stopCtx)
	}()
	// Wait for the app to be stopped.
	<-app.Done()
}
