package command

import (
	"github.com/ds3lab/easeml/client/go/client"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var listUsersCmd = &cobra.Command{
	Use:   "users",
	Short: "Lists users.",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

		if apiKey == "" {
			apiKey = viper.GetString("api-key")
		}
		serverAddress := viper.GetString("server-address")
		context := client.Context{ServerAddress: serverAddress, UserCredentials: client.APIKeyCredentials{APIKey: apiKey}}

		showAll, err := cmd.Flags().GetBool("all")
		if err != nil {
			panic(err)
		}
		status := ""
		if showAll == false {
			status = "active"
		}

		result, err := context.GetUsers(status)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Printf("Number of results: %d\n\n", len(result))

		if len(result) > 0 {
			w := tabwriter.NewWriter(os.Stdout, 0, 8, 3, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tSTATUS")

			for _, r := range result {
				fmt.Fprintf(w, "%s\t%s\t%s\n", r.ID, r.Name, r.Status)
			}

			w.Flush()

		}

	},
}

func init() {
	listCmd.AddCommand(listUsersCmd)

	//loginCmd.PersistentFlags().BoolVarP(&saveAPIKey, "save", "s", false, "Write the resulting API key to the config file.")

	listUsersCmd.Flags().Bool("all", false, "Show both active and archived users.")

}
