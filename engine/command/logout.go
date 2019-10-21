package command

import (
	client "github.com/ds3lab/easeml/client/go/easemlclient"
	"fmt"

	"github.com/spf13/viper"

	"github.com/spf13/cobra"
)

// logoutCmd represents the logout command
var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logs the user out the ease.ml service.",
	Long:  ``,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {

		if apiKey == "" {
			apiKey = viper.GetString("api-key")
		}
		serverAddress := viper.GetString("server-address")
		context := client.Context{ServerAddress: serverAddress, UserCredentials: client.APIKeyCredentials{APIKey: apiKey}}
		err := context.Logout()
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		fmt.Println("Logout successful.")
		return
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)

	logoutCmd.PersistentFlags().String("api-key", "", "API key of the user to log out.")

	viper.BindPFlags(logoutCmd.PersistentFlags())
}
