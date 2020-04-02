# aws-okta-duo

`aws-okta-duo` is a tool that lets you securely access AWS by logging into Okta,
assuming you're using Duo for 2FA.

With `aws-okta-duo`, you'll be able to create convenient AWS automation tools
without sacrificing security:

```bash
# List s3 buckets in the production AWS account.
aws-prod exec -- aws s3 ls

# Open the AWS web console for the prod account.
aws-prod login
```

See ["Recommended usage: `aws-okta-duo` in an
alias"](#recommended-usage-aws-okta-duo-in-an-alias) to see how to do this.

## Installation

Install `aws-okta-duo` by running:

```bash
go install github.com/ucarion/aws-okta-duo
```

## Basic usage

> An important security note:
>
> You can pass `aws-okta-duo` parameters through environment variables (this is
> the *strongly* recommended approach), or through command-line flags (which is
> discouraged). `aws-okta-duo` needs your Okta login credentials to work. You
> should never pass secrets as command arguments, because they'll be leaked to
> anyone running `ps` or some similar program.
>
> Instead, pass secrets through environment variables. Unlike ordinary command
> arguments, environment variables aren't leaked to the process table.

### Running commands within an AWS account

`aws-okta-duo exec` lets you run a command with AWS credentials already
pre-injected. For example, you can run:

```bash
AWS_OKTA_DUO_OKTA_USERNAME="xxx" \
  AWS_OKTA_DUO_OKTA_PASSWORD="xxx" \
  AWS_OKTA_DUO_OKTA_HOST=mycoolcompany.okta.com \
  AWS_OKTA_DUO_OKTA_APP_PATH=/home/my_aws_okta_app_embed_path \
  aws-okta-duo exec -- aws s3 ls
```

Which, once you fill in those environment variables with the appropriate values
(see ["How to find your Okta host and app
path"](#how-to-find-your-okta-host-and-app-path) for guidance on this), you'll
the usual `aws s3 ls` output to appear, within the context of the AWS account
you specified. You'll get a push notification from Duo in the process, too.

Under the hood, this works by executing your command with the [special AWS
environment variables][aws-envvars] pre-injected -- you can see this for
yourself by running `aws-okta-duo exec -- env`:

```bash
AWS_OKTA_DUO_OKTA_USERNAME="xxx" \
  AWS_OKTA_DUO_OKTA_PASSWORD="xxx" \
  AWS_OKTA_DUO_OKTA_HOST=mycoolcompany.okta.com \
  AWS_OKTA_DUO_APP_PATH=/home/my_aws_okta_app_embed_path \
  aws-okta-duo exec -- env
```

Which outputs:

```text
... a bunch of stuff
AWS_ACCESS_KEY_ID=A_TEMPORARY_AWS_ACCESS_KEY_ID
AWS_ACCESS_KEY_SECRET=A_TEMPORARY_AWS_ACCESS_KEY_SECRET
AWS_SESSION_TOKEN=A_TEMPORARY_AWS_SESSION_TOKEN
```

### Opening an AWS account in your web browser

You can also use `aws-okta-okta login` to open a web console session for an
account. For example:

```bash
AWS_OKTA_DUO_OKTA_USERNAME="xxx" \
  AWS_OKTA_DUO_OKTA_PASSWORD="xxx" \
  AWS_OKTA_DUO_OKTA_HOST=mycoolcompany.okta.com \
  AWS_OKTA_DUO_APP_PATH=/home/my_aws_okta_app_embed_path \
  aws-okta-duo login
```

## Advanced usage

### Caching Okta session IDs

If you have 2FA with Duo enabled on Okta, then you need to respond to a push
notification every time you log in. This can be annoying. You can avoid getting
a push notification every time you run `aws-okta-duo` by using a pair of
arguments to `aws-okta-duo`:

* `AWS_OKTA_DUO_OKTA_SESSION_ID` lets you provide an existing Okta session ID
  for `aws-okta-duo` to try to use. If it turns out to be invalid or expired,
  `aws-okta-duo` goes through the entire flow from the beginning.

* `AWS_OKTA_DUO_SAVE_SESSION_CMD` / `--save-session-cmd` lets you provide the
  name of an executable that `aws-okta-duo` should invoke every time it acquires
  a valid Okta session ID. `aws-okta-duo` invokes the given executable with an
  `OKTA_SESSION_ID` environment variable containing the session ID.

Using these two commands together, you can implement a caching mechanism for
Okta session IDs. When you invoke `aws-okta-duo`, pass a `--save-session-cmd`
that writes the Okta session ID to some secure store, such as your operating
system keychain/keyring -- and have your `AWS_OKTA_DUO_OKTA_SESSION_ID` be
populated from that same secure store.

See the next section, on using `aws-okta-duo` in an alias, for an example of how
you can do this in more detail.

## Recommended usage: `aws-okta-duo` in an alias

Unlike commands like [`aws-vault`][aws-vault] or [`aws-okta`][aws-okta],
`aws-okta-duo` does not try to solve everything related to using AWS with
Okta/Duo. Instead, `aws-okta-duo` gives you the essential building blocks for
automating your AWS interactions in the face of difficult-to-automate software
like Okta or Duo.

It's recommended that you build simple scripts on top of `aws-okta-duo` to help
automate access to AWS. In this section, we'll discuss a recommended approach to
building a little `aws-prod` command, which engineers can use to exec commands
in a production AWS context, or log into the production AWS console.

At the end of this section, you'll have a script that can do this:

```bash
# List s3 buckets in the production AWS account.
aws-prod exec -- aws s3 ls

# Open the AWS web console for the prod account.
aws-prod login
```

> The code samples in this section are also available in the `examples` of this
> repo. You'll first need to update those scripts to use your organization's
> Okta domain and app paths.
>
> From the directory that this README is in, you can test them by running:
>
> ```bash
> # First, set up your Okta username/password into a keychain entry:
> ./examples/aws-okta-duo-wrapper-setup.sh
>
> # Now we can run stuff against an AWS account!
> ./examples/aws-okta-duo-wrapper.sh exec -- aws s3 ls
> ./examples/aws-okta-duo-wrapper.sh login
> ```

Writing a command like `aws-prod` requires using a secure password store. We're
going to use MacOS's Keychain for this -- you'll need to modify this example to
make it work on other operating systems.

Let's begin by going in reverse -- here's the the final script we'll create:

```bash
#!/bin/bash
#
# This would go in /usr/local/bin/aws-prod, or some other location on the PATH.

session_id=$(security find-generic-password -s aws-okta-duo-wrapper-okta-session-id -w 2>/dev/null)
username=$(security find-generic-password -s aws-okta-duo-wrapper-okta-username -w)
password=$(security find-generic-password -s aws-okta-duo-wrapper-okta-password -w)

AWS_OKTA_DUO_OKTA_SESSION_ID=$session_id \
  AWS_OKTA_DUO_OKTA_HOST=yourcompany.otka.com \ # fill this in
  AWS_OKTA_DUO_OKTA_USERNAME=$username \
  AWS_OKTA_DUO_OKTA_PASSWORD=$password \
  AWS_OKTA_DUO_OKTA_APP_PATH=/path/to/prod/aws/in/okta \ # fill this in
  AWS_OKTA_DUO_SAVE_SESSION_CMD=aws-okta-duo-wrapper-save-session \
  aws-okta-duo $@
```

This `aws-prod` command just forwards all of the arguments you gave it to
`aws-okta-duo`, but with some environment variables loaded from the MacOS
Keychain (that's what the `security find-generic-password` calls are doing). We
load the Okta username, password, and session ID from the keychain.

You'll notice this script invokes a `aws-okta-duo-wrapper-save-session`. That's
a script that is responsible for saving Okta session IDs, so that we don't have
to do a push notification on every invocation of `aws-prod`. This follows the
recommended technique in ["Caching Okta session
IDs"](#caching-okta-session-ids) above.

Here's an implementation of this `aws-okta-duo-wrapper-save-session` script:

```bash
#!/bin/bash
#
# This would go in /usr/local/bin/aws-okta-duo-wrapper-save-session.

security delete-generic-password -a $USER -s aws-okta-duo-wrapper-okta-session-id > /dev/null 2>&1
security -i <<< "add-generic-password -a $USER -s aws-okta-duo-wrapper-okta-session-id -w '$OKTA_SESSION_ID'"
```

All this script does is read the `OKTA_SESSION_ID` environment variable (see
["Caching Okta session IDs"](#caching-okta-session-ids) for more details on
this), and save it to the same place we load `AWS_OKTA_DUO_OKTA_SESSION_ID` from
in the `aws-prod` command. Together, they implement a secure cache for Otka
session IDs.

To help users store up their Okta username and password in the Keychain in the
place `aws-prod` expects them, you can also provide a setup script:

```bash
#!/bin/bash
#
# This would go in /usr/local/bin/aws-okta-duo-wrapper-setup.

echo -n "okta username: "
read -r username

echo -n "okta password: "
read -s password

security delete-generic-password -a $USER -s aws-okta-duo-wrapper-okta-username > /dev/null 2>&1
security -i <<< "add-generic-password -a $USER -s aws-okta-duo-wrapper-okta-username -w '$username'"

security delete-generic-password -a $USER -s aws-okta-duo-wrapper-okta-password > /dev/null 2>&1
security -i <<< "add-generic-password -a $USER -s aws-okta-duo-wrapper-okta-password -w '$password'"
```

And that's it! Now you can run:

```bash
# List s3 buckets in the production AWS account.
aws-prod exec -- aws s3 ls

# Open the AWS web console for the prod account.
aws-prod login
```

## How to find your Okta host and app path

### Finding your Okta host

When you log into Okta and go to the home of the web version of Okta, you're
probably going to end up on some domain like this:

```text
https://XXX.okta.com/app/UserHome
```

The `XXX.okta.com` is the `AWS_OKTA_DUO_OKTA_HOST` / `--okta-host` you need to
provide to `aws-okta-duo`.

### Finding your Okta app path

To find your app's path, you'll need to be an admin in your organization's Okta.
From the admin view, go to the "Applications" tab and find the AWS connection
you'd like to use through `aws-okta-duo`. Make sure:

* The app is connected to the proper AWS account. Don't accidentally choose the
  production app instead of the dev app.
* The app is using SAML, not OAuth/OIDC. `aws-okta-duo` only supports SAML apps.

With those things verified, go to the "App Embed Link" section, and find the
"Embed Link" subsection. In it is a text area with a URL that looks something
like:

```text
https://XXX.oktapreview.com/home/amazon_aws/YYYYYYYYYYYYYYYYYYYY/ZZZ
```

The `/home/amazon_aws/YYYYYYYYYYYYYYYYYYYY/ZZZ` is the
`AWS_OKTA_DUO_OKTA_APP_PATH` / `--okta-app-path` you need to provide to
`aws-okta-duo`.

[aws-envvars]: https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html
[aws-vault]: https://github.com/99designs/aws-vault
[aws-okta]: https://github.com/segmentio/aws-okta
