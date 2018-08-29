package command

import (
	"github.com/ds3lab/easeml/client"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var listJobUser, listJobStatus, listJobDataset, listJobObjective, listJobModel string

var listJobsCmd = &cobra.Command{
	Use:   "jobs",
	Short: "Lists jobs.",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

		if apiKey == "" {
			apiKey = viper.GetString("api-key")
		}
		serverAddress := viper.GetString("server-address")
		context := client.Context{ServerAddress: serverAddress, UserCredentials: client.APIKeyCredentials{APIKey: apiKey}}

		result, err := context.GetJobs(listJobUser, listJobStatus, listJobDataset, listJobObjective, listJobModel)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Printf("Number of results: %d\n\n", len(result))

		if len(result) > 0 {
			w := tabwriter.NewWriter(os.Stdout, 0, 8, 3, ' ', 0)
			fmt.Fprintln(w, "USER\tDATASET\tOBJECTIVE\tNUM MODELS\tRUNNING TIME\tSTATUS")

			for _, r := range result {
				fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%d\t%s\n", r.User, r.Dataset, r.Objective, len(r.Models), r.RunningDuration-r.PauseDuration, r.Status)
			}

			w.Flush()

		}

	},
}

func init() {
	listCmd.AddCommand(listJobsCmd)

	listJobsCmd.Flags().StringVar(&listJobUser, "user", "", "Filter jobs by user.")
	listJobsCmd.Flags().StringVar(&listJobStatus, "status", "", "Filter jobs by status.")
	listJobsCmd.Flags().StringVar(&listJobDataset, "dataset", "", "Filter jobs by dataset.")
	listJobsCmd.Flags().StringVar(&listJobObjective, "objective", "", "Filter jobs by objective.")
	listJobsCmd.Flags().StringVar(&listJobModel, "model", "", "Show only jobs that contain the given model.")

}
