package command

import (
	"github.com/ds3lab/easeml/client/go/client"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var showUserCmd = &cobra.Command{
	Use:   "user [id]",
	Short: "Shows user given its id. If no id is specified, the current user is shown.",
	Long:  ``,
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {

		if apiKey == "" {
			apiKey = viper.GetString("api-key")
		}
		serverAddress := viper.GetString("server-address")
		context := client.Context{ServerAddress: serverAddress, UserCredentials: client.APIKeyCredentials{APIKey: apiKey}}

		id := ""
		if len(args) > 0 {
			id = args[0]
		}

		result, err := context.GetUserByID(id)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 8, 3, ' ', 0)

		fmt.Fprintf(w, "ID:\t%s\n", result.ID)
		fmt.Fprintf(w, "NAME:\t%s\n", result.Name)
		fmt.Fprintf(w, "STATUS:\t%s\n", result.Status)

		w.Flush()

	},
}

func init() {
	showCmd.AddCommand(showUserCmd)
}
