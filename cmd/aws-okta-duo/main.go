package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ucarion/aws-okta-duo/internal/provider"
	"golang.org/x/sys/unix"
)

var (
	rootCmd = &cobra.Command{
		Use:   "aws-okta-duo",
		Short: "A CLI tool that automates creating AWS sessions via Okta + Duo",
	}

	execCmd = &cobra.Command{
		Use:  "exec -- [cmd...]",
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			do(func() error {
				stsClient := sts.New(session.Must(session.NewSession()))
				httpClient := &http.Client{}
				provider := provider.Provider{
					OktaSessionID: viper.GetString("okta-session-id"),
					OktaHost:      viper.GetString("okta-host"),
					OktaUsername:  viper.GetString("okta-username"),
					OktaPassword:  viper.GetString("okta-password"),
					OktaAppPath:   viper.GetString("okta-app-path"),
					DuoDevice:     viper.GetString("duo-device"),
					STSClient:     stsClient,
					HTTPClient:    httpClient,
				}

				res, err := provider.GetCredentials(cmd.Context())
				if err != nil {
					return err
				}

				saveSessionCmd := viper.GetStringSlice("save-session-cmd")
				if len(saveSessionCmd) > 0 {
					cmd := exec.CommandContext(cmd.Context(), saveSessionCmd[0], saveSessionCmd[1:]...)
					cmd.Env = os.Environ()
					cmd.Env = append(cmd.Env, fmt.Sprintf("OKTA_SESSION_ID=%s", res.OktaSessionID))

					if err := cmd.Start(); err != nil {
						return err
					}
				}

				env := []string{}
				for _, e := range os.Environ() {
					if !strings.HasPrefix(e, "AWS_OKTA_DUO") {
						env = append(env, e)
					}
				}

				env = append(env, fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", *res.Credentials.AccessKeyId))
				env = append(env, fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", *res.Credentials.SecretAccessKey))
				env = append(env, fmt.Sprintf("AWS_SESSION_TOKEN=%s", *res.Credentials.SessionToken))

				argv0, err := exec.LookPath(args[0])
				if err != nil {
					return err
				}

				return unix.Exec(argv0, args[0:], env)
			})
		},
	}
)

func init() {
	rootCmd.AddCommand(execCmd)

	rootCmd.PersistentFlags().String("okta-session-id", "", "An existing Okta session ID to try to use instead of authenticating")
	rootCmd.PersistentFlags().String("okta-host", "", "The host that your Okta organization is served from (required)")
	rootCmd.PersistentFlags().String("okta-username", "", "Your Okta username (required)")
	rootCmd.PersistentFlags().String("okta-password", "", "Your Okta password (required)")
	rootCmd.PersistentFlags().String("okta-app-path", "", "The embed URL of the Okta app to log into (required)")
	rootCmd.PersistentFlags().String("duo-device", "phone1", "The device that Duo pushes should be sent to")
	rootCmd.PersistentFlags().StringSlice("save-session-cmd", nil, "A process to run after acquiring a new Okta session ID.")

	viper.SetEnvPrefix("AWS_OKTA_DUO")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
	viper.BindPFlags(rootCmd.PersistentFlags())
}

func main() {
	do(rootCmd.Execute)
}

func do(f func() error) {
	if err := f(); err != nil {
		fmt.Fprintln(os.Stdout, err)
		os.Exit(1)
	}
}
