# Assume IAM Roles using the Hub-Spoke model

When you have _several_ AWS accounts to manage, you can keep things secure and locked-down by adopting the hub-spoke model of assuming IAM roles across accounts.

1. You have a user (or a bot if you're automating) with permission to assume the "Hub" role in an account (doesn't need to be the same as the user).

1. From the "Hub", the user can then traverse to a "Spoke" account to perform the actions that are granted to an assumer of that "Hub" role.

This model is recommended by AWS (read below). You will need to provision the roles (via [Service Control Policies](https://docs.aws.amazon.com/organizations/latest/userguide/orgs_manage_policies_scps.html) or perhaps [Terraform](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role)), but there's only one user (per workload) that you need to grant permissions to.

> [!IMPORTANT]
> This is definitely more _complex_ than the older approaches like creating an IAM user in every account, then sharing those credentials with the whole team. However, that older approach is difficult to keep _secured_, and needs to be rotated every time someone leaves the team (otherwise they will continue to have access).
>
> By leveraging IAM role assumption, and designating a single hub account that can speak to 1+ spoke toles, you create secure barriers between accounts with fewer things to manage.

## What is the Hub-Spoke model?

In a multi-account setup, optionally managed with AWS Organizations and AWS Identity Center, think of it like a bicycle wheel: One hub, many spokes.

For example, if an automated process (e.g., a automation) needs to perform the same kinds of actions in every account you own (e.g., security analysis, reporting, account inventory), you would set up:

1. A "service account" for the automation/job/process/whatever.
1. Designate one account as the "hub". You need to connect here before connecting to anything else. It is conceptually similar to a "jump box".
1. All accounts where actions need to be performed are the "spokes". They would all have the same policy that can be assumed by the user connecting through the hub account and out to spoke accounts.

In larger setups that use AWS Organizations, these policies can be provisioned with _Service Control Policies_ (SCPs). In smaller setups, you can use tools like Terraform or the AWS CLI for automation.

## Install as a CLI tool

1. You must have the Golang toolchain installed first.

    ```bash
    brew install go
    ```

1. Add `$GOPATH/bin` to your `$PATH` environment variable. By default (i.e., without configuration), `$GOPATH` is defined as `$HOME/go`.

    ```bash
    export PATH="$PATH:$GOPATH/bin"
    ```

1. Once you've done everything above, you can use `go install`.

    ```bash
    go install github.com/northwood-labs/assume-spoke-role@latest
    ```

## Usage as CLI Tool

```bash
# Learn how it works.
assume-spoke-role --help
```

Run a command in another account (assuming you have permissions to assume a role). The ` -- ` marker signifies the end of passing options, and to begin treating subsequent text as the command to run with those credentials.

Assuming you're using [AWS Vault](https://github.com/99designs/aws-vault) to manage your credentials, and want to manage common configurations via environment variables:

```bash
# Only if you need this.
export ASSUME_ROLE_EXTERNAL_ID="this-is-my-automation"

# Optional, but recommended.
export ASSUME_ROLE_SESSION_STRING="me@example.com"

# Pre-configure which things to connect to.
export ASSUME_ROLE_HUB_ACCOUNT="999999999999"
export ASSUME_ROLE_HUB_ROLE="automation-hub-role"
export ASSUME_ROLE_SPOKE_ROLE="automation-spoke-role"

# Using your local credentials (e.g., sys-automation), assume a role in the "HUB"
# account, before pivoting to a "SPOKE" account, then executing a command with
# those "SPOKE" credentials.
aws-vault exec sys-automation -- \
    assume-spoke-role run --spoke-account 888888888888 -- \
        aws sts get-caller-identity
```

Or, if you want to more explicitly rely on CLI parameters rather than environment variables:

```bash
aws-vault exec sys-automation -- \
    assume-spoke-role run \
        --hub-account 999999999999 \
        --spoke-account 888888888888 \
        --hub-role "automation-hub-role" \
        --spoke-role "automation-spoke-role" \
        --external-id "this-is-my-automation" \
        -- \
            aws sts get-caller-identity
```

## Usage as Library

This can also be used as a library in your own applications for generating a set of STS credentials.

```go
import (
  "github.com/northwood-labs/assume-spoke-role/hubspoke"
  "github.com/northwood-labs/awsutils"
)

func main() {
    // Get AWS credentials from environment.
    config, err := awsutils.GetAWSConfig(ctx, "", "", 3, false)
    if err != nil {
        log.Fatal(fmt.Sprintf("could not generate a valid AWS configuration object: %w", err))
    }

    // Assume appropriate roles and return session credentials for the "Spoke" account.
    roleCredentials, _, err := hubspoke.GetSpokeCredentials(&hubspoke.SpokeCredentialsInput{
        Context:        ctx,
        Config:         &config,
        HubAccountID:   "888888888888",
        SpokeAccountID: "999999999999",
        HubRoleName:    "hub-role",
        SpokeRoleName:  "spoke-role",
        ExternalID:     "this-is-my-automation", // Only if you need this.
        SessionString:  "me@example.com",        // Optional.
    })
    if err != nil {
        log.Fatal(fmt.Sprintf("could not generate valid AWS credentials for the 'spoke' account: %w", err))
    }

    fmt.Printf("AWS_ACCESS_KEY_ID=%s\n", *roleCredentials.AccessKeyId),
    fmt.Printf("AWS_SECRET_ACCESS_KEY=%s\n", *roleCredentials.SecretAccessKey),
    fmt.Printf("AWS_SECURITY_TOKEN=%s\n", *roleCredentials.SessionToken),
    fmt.Printf("AWS_SESSION_TOKEN=%s\n", *roleCredentials.SessionToken),
    fmt.Printf("AWS_SESSION_EXPIRATION=%s\n", roleCredentials.Expiration.String()),
}
```

See `cmd/run.go`, which implements this library to produce this very same CLI tool.

## Setting up the Hub/Spoke configuration

Following the [Principle of Least Privilege](https://www.cisecurity.org/spotlight/ei-isac-cybersecurity-spotlight-principle-of-least-privilege/), we're going to scope-down the permissions to as few as necessary.

### The User

In one of your AWS accounts, create an IAM user/instance-profile/whatever dedicated to this task. Since this user represents a _process_ and not a _person_, I recommend prefixing the user name with `sys-`. If we wanted to do things on behalf of the "automation" process, then perhaps we'd call this user `sys-automation`.

> [!TIP]
> If this is a human user, then using AWS Organizations and AWS Identity Center (née AWS SSO) is recommended as a starting (bootstrapping) user.

Just like a [Meeseeks](https://rickandmorty.fandom.com/wiki/Mr._Meeseeks), this user is intended for only a single task. It's better to have more users (with corresponding spoke roles) with fewer permissions, than it is to have fewer users (with corresponding spoke roles) with more permissions. Please don't re-use the same user for many tasks, as you increase your cybersecurity "blast radius" that way.

This user — as itself — can only do one thing: assume an IAM role in the "hub" account. (Replace `{hub-account-id}` with the AWS Account ID where your "hub" role is located.)

> [!TIP]
> If you're using AWS Identity Center (née AWS SSO), make sure that the base role you're assuming (e.g., `*-PowerUserAccess`) is authorized to perform `sts:AssumeRole` on your "hub role".

#### Policy

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "sts:AssumeRole",
      "Resource": [
        "arn:aws:iam::{hub-account-id}:role/automation-hub-role"
      ]
    }
  ]
}
```

#### AWS Vault

Using [AWS Vault](https://github.com/99designs/aws-vault), this stores them in the system keychain instead of as plain text on-disk. It automatically generates STS session credentials on your behalf, and it's easy to pass the credentials to things that are built with the AWS SDKs _besides_ the AWS CLI. Oh — and it also supports AWS Identity Center out-of-the-box.

### The Hub

Using the "automation" process example, let's follow-through with creating an IAM role to assume, and call it `automation-hub-role`. This is an IAM role which will grant access to your user for assuming the "spoke" role in every account which has that identically-named role.

> [!TIP]
> If you're not using AWS Organizations, you can remove the entire `Condition` block. You should also specify the AWS Account IDs of the accounts you want to access.)

#### Policy

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "sts:AssumeRole"
      ],
      "Resource": "arn:aws:iam::*:role/automation-spoke-role",
      "Condition": {
        "StringEquals": {
          "aws:PrincipalOrgID": "o-ZZZZZZZZZZ"
        }
      }
    }
  ]
}

```

You'll also need to configure the _trust relationship_ for the "hub" role so that only our user can assume it.

> [!TIP]
> If you're not using AWS Organizations, you can remove the entire `Condition` block.

#### Trust Relationship for a single IAM user

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::{user-creation-account-id}:user/sys-automation"
      },
      "Action": "sts:AssumeRole",
      "Condition": {
        "StringEquals": {
          "aws:PrincipalOrgID": "o-ZZZZZZZZZZ"
        }
      }
    }
  ]
}
```

#### Trust Relationship for an AWS Identity Center (née AWS SSO) user

> [!NOTE]
> The following example assumes that your AWS Identity Center roles (`AWSReservedSSO*`) were configured using the same naming pattern as is suggested by default. If they didn't follow the same pattern, you'll need to adapt the example below.

This will allow any SSO user (in your AWS Organizations account) which is able to assume the `AWSReservedSSO_*-AdministratorAccess_*` or `AWSReservedSSO_*-PowerUserAccess_*` roles to assume the "hub role".

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::{hub-account-id}:root"
      },
      "Action": "sts:AssumeRole",
      "Condition": {
        "ArnLike": {
          "aws:PrincipalARN": [
            "arn:aws:iam::{hub-account-id}:role/aws-reserved/sso.amazonaws.com/us-east-2/AWSReservedSSO_*-AdministratorAccess_*",
            "arn:aws:iam::{hub-account-id}:role/aws-reserved/sso.amazonaws.com/us-east-2/AWSReservedSSO_*-PowerUserAccess_*"
          ]
        },
        "StringEquals": {
          "aws:PrincipalOrgID": "o-ZZZZZZZZZZ"
        }
      }
    }
  ]
}
```

This creates a **bi-directional symbiosis** where the user can only assume the hub role, and the hub role can only be assumed by the user.

<!-- Graphic to explain better? -->

### The Spoke

Using the "automation" process example, let's follow-through with creating an IAM role to assume, and call it `automation-spoke-role`.

This is an IAM role which will grant access to your user (via the hub role) and grants the permissions for what can be done in this account. In our case, we want to grant `ReadOnlyAccess` (the built-in, AWS managed policy). Your needs may be different, so adapt accordingly.

You will need to configure the _trust relationship_ for the "spoke" role so that only our "hub" role can access it.

For an extra bit of entropy in our security, we can require an _External ID_ which is known only to the IAM role and the user accessing it.

#### Policy

This should be a policy which lists the things that the assuming user is permitted to do once they've successfully assumed the spoke account.

#### Trust Relationship

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::{hub-account-id}:role/automation-hub-role"
      },
      "Action": "sts:AssumeRole",
      "Condition": {
        "StringEquals": {
          "aws:PrincipalOrgID": "o-ZZZZZZZZZZ"
        }
      }
    }
  ]
}
```

This creates a **bi-directional symbiosis** where the only action that the "hub role" can perform is to assume the "spoke role", and the spoke role can only be assumed by the hub role.
