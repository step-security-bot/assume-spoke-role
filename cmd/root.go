// Copyright 2019-2024, Northwood Labs
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

const defaultAWSRetries = 3

var (
	fDebug   bool
	fRetries int
	fVerbose bool

	// rootCmd represents the base command when called without any subcommands
	rootCmd = &cobra.Command{
		TraverseChildren: true,
		Use:              "assume-spoke-role",
		Short:            "Assume an IAM hub role, then IAM spoke role, in AWS.",
		Long: `--------------------------------------------------------------------------------
assume-spoke-role

Assume an IAM hub role, then IAM spoke role, in AWS.

This CLI tool is useful when you have multiple AWS accounts, and need to connect
across them using an AWS-recommended hub-and-spoke model. This is when you
designate a single AWS account as a "hub account", and then designate other
accounts as "spoke accounts".

You only need to manage a single inbound IAM credential that is able to traverse
across multiple accounts, instead of multiple credentials that you then have to
manage and secure.

Once you've set-up your roles correctly (which this package does not help with),
you can begin with a base role (e.g., an IAM user, an AWS Identity Center (SSO)
role), then assume a "hub role" in the hub account, then assume a "spoke role"
in the spoke account.
--------------------------------------------------------------------------------`,
	}
)

func init() { // lint:allow_init
	rootCmd.Flags().BoolVarP(
		&fDebug,
		"debug",
		"d",
		false,
		"Run with support for Go debuggers like delve.",
	)

	rootCmd.Flags().IntVarP(
		&fRetries,
		"retries",
		"r",
		defaultAWSRetries,
		"Number of times to retry AWS API calls.",
	)

	rootCmd.Flags().BoolVarP(
		&fVerbose,
		"verbose",
		"v",
		false,
		"Enable verbose logging. This includes over-the-wire AWS SDK logging.",
	)
}

// Execute configures the Cobra CLI app framework and executes the root command.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
