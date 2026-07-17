// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

// Package main is the entry point for the Terraform provider.
//
// This file initializes and serves the VayuCloud Terraform provider.
// It follows the standard pattern for Terraform Plugin Framework providers.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider"
)

// version is set during build time via ldflags
// Example: go build -ldflags="-X main.version=1.0.0"
var version = "dev"

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		// Address is the full address of the provider, including the registry namespace.
		// This is used when the provider is fetched from a registry.
		// Format: registry.terraform.io/<namespace>/<provider-name>
		Address: "registry.terraform.io/tatacommunicationsvayu/vayucloud",
		Debug:   debug,
	}

	// Serve the provider. This blocks until the provider is stopped.
	err := providerserver.Serve(context.Background(), provider.New(version), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
