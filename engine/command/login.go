package command

import (
	"github.com/ds3lab/easeml/client/go/easemlclient"
	"fmt"

	"github.com/howeyc/gopass"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var saveAPIKey bool

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login username",
	Short: "Logs the user into the ease.ml service and returns an API key.",
	Long:  ``,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		username := args[0]
		fmt.Printf("Password: ")
		password, err := gopass.GetPasswdMasked()
		if err != nil {
			fmt.Printf(err.Error() + "\n")
			return
		}
		serverAddress := viper.GetString("server-address")
		context := client.Context{ServerAddress: serverAddress}
		apiKey, err := context.Login(username, string(password))
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Printf("API KEY: %s\n", apiKey)

		if saveAPIKey {
			err := updateConfigAndWrite(map[string]interface{}{"api-key": apiKey})
			if err != nil {
				panic(err)
			}
		}

	},
}

func init() {
	rootCmd.AddCommand(loginCmd)

	loginCmd.PersistentFlags().BoolVarP(&saveAPIKey, "save", "s", false, "Write the resulting API key to the config file.")

}
