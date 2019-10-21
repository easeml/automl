package command

import (
	client "github.com/ds3lab/easeml/client/go/easemlclient"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var showJobCmd = &cobra.Command{
	Use:   "job id",
	Short: "Shows job given its id.",
	Long:  ``,
	Args:  cobra.ExactArgs(1),
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

		result, err := context.GetJobByID(id)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 8, 3, ' ', 0)

		fmt.Fprintf(w, "ID:\t%s\n", result.ID)
		fmt.Fprintf(w, "USER:\t%s\n", result.User)
		fmt.Fprintf(w, "DATASET:\t%s\n", result.Dataset)
		fmt.Fprintf(w, "OBJECTIVE:\t%s\n", result.Objective)
		fmt.Fprintf(w, "MODELS:\n")
		for i := range result.Models {
			fmt.Fprintf(w, "  - %s\n", result.Models[i])
		}
		if result.AcceptNewModels {
			fmt.Fprintf(w, "ACCEPT NEW MODELS:\tYES\n")
		} else {
			fmt.Fprintf(w, "ACCEPT NEW MODELS:\tNO\n")
		}

		if len(result.AltObjectives) > 0 {
			fmt.Fprintf(w, "ALTERNATIVE OBECTIVES:\n")
			for i := range result.AltObjectives {
				fmt.Fprintf(w, "  - %s\n", result.AltObjectives[i])
			}
		}
		fmt.Fprintf(w, "STATUS:\t%s\n", result.Status)
		if result.StatusMessage != "" {
			fmt.Fprintf(w, "STATUS MESSAGE:\t%s\n", result.StatusMessage)
		}
		fmt.Fprintf(w, "CREATION TIME:\t%s\n", result.CreationTime)
		fmt.Fprintf(w, "RUNNING DURATION:\t%d\n", result.RunningDuration-result.PauseDuration)
		fmt.Fprintf(w, "PAUSE DURATION:\t%d\n", result.PauseDuration)
		w.Flush()

	},
}

func init() {
	showCmd.AddCommand(showJobCmd)
}
