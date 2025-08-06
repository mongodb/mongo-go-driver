// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"log"

	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{
		Use:     "perfcomp",
		Short:   "perfcomp is a cli that reports stat-sig results between evergreen patches with the mainline commit",
		Version: "0.0.0-alpha",
	}

	cmd.AddCommand(newCompareCommand())
	cmd.AddCommand(newMdCommand())

	if err := cmd.Execute(); err != nil {
		log.Fatalf("error: %v", err)
	}
}
