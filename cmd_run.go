package main

import (
	"os"

	errors "github.com/go-faster/errors"
	cli "github.com/jawher/mow.cli"
	"github.com/northwood-labs/awsutils"
	"github.com/northwood-labs/golang-utils/exiterrorf"

	"github.com/northwood-labs/assume-spoke-role/hubspoke"
)

func cmdRun(cmd *cli.Cmd) {
	cmd.LongDesc = `Perform the action of assuming roles and running an action.

Use environment variables to store parameter values consistently. CLI options
take precedence over environment variables.`

	const (
		defaultAWSRetries = 3
	)

	var (
		externalID    = os.Getenv("ASSUME_ROLE_EXTERNAL_ID")
		hubAccount    = os.Getenv("ASSUME_ROLE_HUB_ACCOUNT")
		hubRole       = os.Getenv("ASSUME_ROLE_HUB_ROLE")
		spokeAccount  = os.Getenv("ASSUME_ROLE_SPOKE_ACCOUNT")
		spokeRole     = os.Getenv("ASSUME_ROLE_SPOKE_ROLE")
		sessionString = os.Getenv("ASSUME_ROLE_SESSION_STRING")

		externalIDFlag = cmd.StringOpt(
			"e external-id",
			externalID,
			"(ASSUME_ROLE_EXTERNAL_ID) The external ID value that is required by your hub and spoke policies, if any.",
		)
		hubAccountFlag = cmd.StringOpt(
			"h hub-account",
			hubAccount,
			"(ASSUME_ROLE_HUB_ACCOUNT) The 12-digit AWS account ID containing the HUB policy.",
		)
		spokeAccountFlag = cmd.StringOpt(
			"s spoke-account",
			spokeAccount,
			"(ASSUME_ROLE_SPOKE_ACCOUNT) The 12-digit AWS account ID containing the SPOKE policy.",
		)
		hubRoleFlag = cmd.StringOpt(
			"H hub-role",
			hubRole,
			"(ASSUME_ROLE_HUB_ROLE) The name of the IAM role to assume in the HUB account.",
		)
		spokeRoleFlag = cmd.StringOpt(
			"S spoke-role",
			spokeRole,
			"(ASSUME_ROLE_SPOKE_ROLE) The name of the IAM role to assume in the HUB account.",
		)
		sessionStringFlag = cmd.StringOpt(
			"session-string",
			sessionString,
			"(ASSUME_ROLE_SESSION_STRING) A string that will be part of the resulting User ID in the spoke account.",
		)
		retriesFlag = cmd.IntOpt(
			"r retries",
			defaultAWSRetries,
			"The maximum number of retries that the underlying AWS SDK should perform.",
		)
		verboseFlag = cmd.BoolOpt(
			"v verbose",
			false,
			"Enable verbose logging.",
		)
		cmdd = cmd.StringsArg("COMMAND", []string{""}, "The command to run using the spoke policy.")
	)

	cmd.Spec = `[-e=<external-id>] [-h=<hub-account>] [-s=<spoke-account>] [-H=<hub-role>] ` +
		`[-S=<spoke-role>] [-r=<retries>] [--verbose] -- COMMAND...`

	cmd.Action = func() {
		// Get AWS credentials from environment.
		config, err := awsutils.GetAWSConfig(ctx, "", "", *retriesFlag, *verboseFlag)
		if err != nil {
			exiterrorf.ExitErrorf(errors.Wrap(err, "could not generate a valid AWS configuration object"))
		}

		// Assume appropriate roles and return session credentials for the "Spoke" account.
		roleCredentials, _, err := hubspoke.GetSpokeCredentials(&hubspoke.SpokeCredentialsInput{
			Context:        ctx,
			Config:         &config,
			HubAccountID:   *hubAccountFlag,
			SpokeAccountID: *spokeAccountFlag,
			HubRoleName:    *hubRoleFlag,
			SpokeRoleName:  *spokeRoleFlag,
			ExternalID:     *externalIDFlag,
			SessionString:  *sessionStringFlag,
		})
		if err != nil {
			exiterrorf.ExitErrorf(errors.Wrap(err, "could not generate valid AWS credentials for the 'spoke' account"))
		}

		// Pass the spoke credentials to a CLI task.
		runCommand(roleCredentials, *cmdd)
	}
}
