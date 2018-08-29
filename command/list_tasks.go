package command

import (
	"github.com/ds3lab/easeml/client"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var listTaskJob, listTaskUser, listTaskStatus, listTaskStage, listTaskDataset, listTaskObjective, listTaskModel string

var listTasksCmd = &cobra.Command{
	Use:   "tasks",
	Short: "Lists tasks.",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

		if apiKey == "" {
			apiKey = viper.GetString("api-key")
		}
		serverAddress := viper.GetString("server-address")
		context := client.Context{ServerAddress: serverAddress, UserCredentials: client.APIKeyCredentials{APIKey: apiKey}}

		result, err := context.GetTasks(listTaskJob, listTaskUser, listTaskStatus, listTaskStage, listTaskDataset, listTaskObjective, listTaskModel)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Printf("Number of results: %d\n\n", len(result))

		if len(result) > 0 {
			w := tabwriter.NewWriter(os.Stdout, 0, 8, 3, ' ', 0)
			fmt.Fprintln(w, "ID\tMODEL\tOBJECTIVE\tDATASET\tSTATUS\tSTAGE\tRUNNING TIME\tQUALIY")

			for _, r := range result {
				// The running duraion is given in milliseconds.
				runningDuration := time.Duration(r.RunningDuration) * time.Millisecond

				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%f\n", r.ID, r.Model, r.Dataset, r.Objective, r.Status, r.Stage, runningDuration, r.Quality)
			}

			w.Flush()

		}

	},
}

func init() {
	listCmd.AddCommand(listTasksCmd)

	listTasksCmd.Flags().StringVar(&listTaskJob, "job", "", "Filter tasks by job.")
	listTasksCmd.Flags().StringVar(&listTaskUser, "user", "", "Filter tasks by user.")
	listTasksCmd.Flags().StringVar(&listTaskStatus, "status", "", "Filter tasks by status.")
	listTasksCmd.Flags().StringVar(&listTaskStage, "stage", "", "Filter tasks by stage.")
	listTasksCmd.Flags().StringVar(&listTaskDataset, "dataset", "", "Filter tasks by dataset.")
	listTasksCmd.Flags().StringVar(&listTaskObjective, "objective", "", "Filter tasks by objective.")
	listTasksCmd.Flags().StringVar(&listTaskModel, "model", "", "Show only tasks that contain the given model.")

}
