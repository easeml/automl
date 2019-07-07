package command

import (
	"fmt"

	client "github.com/ds3lab/easeml/client/go/easemlclient"

	"github.com/howeyc/gopass"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var userID, userName string

var createUserCmd = &cobra.Command{
	Use:   "user",
	Short: "Creates a new user.",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

		if apiKey == "" {
			apiKey = viper.GetString("api-key")
		}
		serverAddress := viper.GetString("server-address")
		context := client.Context{ServerAddress: serverAddress, UserCredentials: client.APIKeyCredentials{APIKey: apiKey}}

		// User ID is required.
		for userID == "" {
			err := readLine("User ID: ", &userID)
			if err != nil {
				fmt.Printf(err.Error() + "\n")
				return
			}
		}

		// User Name is optional.
		if userName == "" {
			err := readLine("User Name [optional]: ", &userName)
			if err != nil {
				fmt.Printf(err.Error() + "\n")
				return
			}
		}

		fmt.Printf("Password: ")
		password, err := gopass.GetPasswdMasked()
		if err != nil {
			fmt.Printf(err.Error() + "\n")
			return
		}

		result, err := context.CreateUser(userID, string(password), userName)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Printf("SUCCESS: User \"%s\" created.\n", result)

	},
}

func init() {
	createCmd.AddCommand(createUserCmd)

	createUserCmd.Flags().StringVar(&userID, "id", "", "User ID.")
	createUserCmd.Flags().StringVar(&userName, "name", "", "User full name.")

}
