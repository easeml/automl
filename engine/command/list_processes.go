package command

import (
	"fmt"
	"os"
	"text/tabwriter"

	client "github.com/ds3lab/easeml/client/go/easemlclient"
	"github.com/ds3lab/easeml/client/go/easemlclient/types"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var listProcessesCmd = &cobra.Command{
	Use:   "processes",
	Short: "Lists processes.",
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

		tempResult, err := context.GetProcesses("")
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		result := []types.Process{}
		if showAll == false {
			for i := range tempResult {
				if tempResult[i].Status != "terminated" {
					result = append(result, tempResult[i])
				}
			}
		} else {
			result = tempResult
		}

		fmt.Printf("Number of results: %d\n\n", len(result))

		if len(result) > 0 {
			w := tabwriter.NewWriter(os.Stdout, 0, 8, 3, ' ', 0)
			fmt.Fprintln(w, "TYPE\tHOST ID\tHOST ADDRESS\tPROCESSID\tSTATUS\tCREATION TIME")

			for _, r := range result {
				fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n", r.Type, r.HostID, r.HostAddress, r.ProcessID, r.Status, r.StartTime.String())
			}

			w.Flush()

		}

	},
}

func init() {
	listCmd.AddCommand(listProcessesCmd)

	//loginCmd.PersistentFlags().BoolVarP(&saveAPIKey, "save", "s", false, "Write the resulting API key to the config file.")

	listProcessesCmd.Flags().Bool("all", false, "Show both running and terminated processes.")

}
