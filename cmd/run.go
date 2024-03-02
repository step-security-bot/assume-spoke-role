// Copyright 2019-2024, Northwood Labs
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/northwood-labs/awsutils"
	"github.com/northwood-labs/golang-utils/exiterrorf"

	"github.com/aws/aws-sdk-go-v2/service/sts/types"
	"github.com/go-faster/errors"
	"github.com/northwood-labs/assume-spoke-role/hubspoke"
	"github.com/spf13/cobra"
)

var (
	ctx = context.Background()

	fExternalID    string
	fHubAccount    string
	fHubRole       string
	fSessionString string
	fSpokeAccount  string
	fSpokeRole     string

	externalID    = os.Getenv("ASSUME_ROLE_EXTERNAL_ID")
	hubAccount    = os.Getenv("ASSUME_ROLE_HUB_ACCOUNT")
	hubRole       = os.Getenv("ASSUME_ROLE_HUB_ROLE")
	sessionString = os.Getenv("ASSUME_ROLE_SESSION_STRING")
	spokeAccount  = os.Getenv("ASSUME_ROLE_SPOKE_ACCOUNT")
	spokeRole     = os.Getenv("ASSUME_ROLE_SPOKE_ROLE")

	// runCmd represents the run command
	runCmd = &cobra.Command{
		Use:   "run",
		Short: "Run a command using the spoke policy",
		Long: `Perform the action of assuming roles and running an action.

Use environment variables to store parameter values consistently. CLI options
take precedence over environment variables.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Get AWS credentials from environment.
			config, err := awsutils.GetAWSConfig(ctx, "", "", fRetries, fVerbose)
			if err != nil {
				exiterrorf.ExitErrorf(errors.Wrap(err, "could not generate a valid AWS configuration object"))
			}

			// Assume appropriate roles and return session credentials for the "Spoke" account.
			roleCredentials, _, err := hubspoke.GetSpokeCredentials(&hubspoke.SpokeCredentialsInput{
				Context:        ctx,
				Config:         &config,
				HubAccountID:   fHubAccount,
				SpokeAccountID: fSpokeAccount,
				HubRoleName:    fHubRole,
				SpokeRoleName:  fSpokeRole,
				ExternalID:     fExternalID,
				SessionString:  fSessionString,
			})
			if err != nil {
				exiterrorf.ExitErrorf(
					errors.Wrap(err, "could not generate valid AWS credentials for the 'spoke' account"),
				)
			}

			// Pass the spoke credentials to a CLI task.
			runCommand(roleCredentials, args)
		},
	}
)

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringVarP(
		&fExternalID,
		"external-id",
		"e",
		externalID,
		"(ASSUME_ROLE_EXTERNAL_ID) The external ID value that is required by your hub and spoke policies, if any.",
	)

	runCmd.Flags().StringVarP(
		&fHubAccount,
		"hub-account",
		"a",
		hubAccount,
		"(ASSUME_ROLE_HUB_ACCOUNT) The 12-digit AWS account ID containing the HUB policy. [Required]",
	)

	runCmd.Flags().StringVarP(
		&fSpokeAccount,
		"spoke-account",
		"s",
		spokeAccount,
		"(ASSUME_ROLE_SPOKE_ACCOUNT) The 12-digit AWS account ID containing the SPOKE policy. [Required]",
	)

	runCmd.Flags().StringVarP(
		&fHubRole,
		"hub-role",
		"H",
		hubRole,
		"(ASSUME_ROLE_HUB_ROLE) The name of the IAM role to assume in the HUB account. [Required]",
	)

	runCmd.Flags().StringVarP(
		&fSpokeRole,
		"spoke-role",
		"S",
		spokeRole,
		"(ASSUME_ROLE_SPOKE_ROLE) The name of the IAM role to assume in the HUB account. [Required]",
	)

	runCmd.Flags().StringVarP(
		&fSessionString,
		"session-string",
		"I",
		sessionString,
		"(ASSUME_ROLE_SESSION_STRING) A string that will be part of the resulting User ID in the spoke account.",
	)

	var err error

	if hubAccount == "" {
		err = runCmd.MarkFlagRequired("hub-account")
		if err != nil {
			exiterrorf.ExitErrorf(errors.Wrap(err, "could not mark the hub-account flag as required"))
		}
	}

	if hubRole == "" {
		err = runCmd.MarkFlagRequired("hub-role")
		if err != nil {
			exiterrorf.ExitErrorf(errors.Wrap(err, "could not mark the hub-role flag as required"))
		}
	}

	if spokeAccount == "" {
		err = runCmd.MarkFlagRequired("spoke-account")
		if err != nil {
			exiterrorf.ExitErrorf(errors.Wrap(err, "could not mark the spoke-account flag as required"))
		}
	}

	if spokeRole == "" {
		err = runCmd.MarkFlagRequired("spoke-role")
		if err != nil {
			exiterrorf.ExitErrorf(errors.Wrap(err, "could not mark the spoke-role flag as required"))
		}
	}
}

func runCommand(creds *types.Credentials, args []string) {
	cmd := exec.Command(args[0], args[1:]...) // lint:allow_possible_insecure

	cmd.Env = os.Environ()
	cmd.Env = append(
		cmd.Env,
		fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", *creds.AccessKeyId),
		fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", *creds.SecretAccessKey),
		fmt.Sprintf("AWS_SECURITY_TOKEN=%s", *creds.SessionToken),
		fmt.Sprintf("AWS_SESSION_TOKEN=%s", *creds.SessionToken),
		fmt.Sprintf("AWS_SESSION_EXPIRATION=%s", creds.Expiration.String()),
	)

	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
