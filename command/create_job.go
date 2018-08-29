package command

import (
	"github.com/ds3lab/easeml/client"
	"github.com/ds3lab/easeml/database/model"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var jobDataset, jobObjective string
var jobModels, jobAltObjectives []string
var jobAcceptNewModels bool
var jobMaxTasks uint64

var createJobCmd = &cobra.Command{
	Use:   "job",
	Short: "Creates a new job.",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

		if apiKey == "" {
			apiKey = viper.GetString("api-key")
		}
		serverAddress := viper.GetString("server-address")
		context := client.Context{ServerAddress: serverAddress, UserCredentials: client.APIKeyCredentials{APIKey: apiKey}}

		// Job dataset is required.
		for jobDataset == "" {
			err := readLine("Job Dataset: ", &jobDataset)
			if err != nil {
				fmt.Printf(err.Error() + "\n")
				return
			}
		}

		// Try to get the dataset.
		dataset, err := context.GetDatasetByID(jobDataset)
		if err != nil {
			fmt.Println("Error: " + err.Error())
			return
		}

		// Find all applicable objectives.
		// The input schema of the objective must correspond to the output schema of the dataset.
		objectives, err := context.GetModules(model.ModuleObjective, "", model.ModuleActive, "", dataset.SchemaOut, "")
		if err != nil {
			fmt.Println("Error: " + err.Error())
			return
		}
		if len(objectives) == 0 {
			fmt.Println("Error: No objectives applicable to the given dataset.")
			return
		}

		if jobObjective == "" || len(jobAltObjectives) == 0 {
			fmt.Println("APPLICABLE OBJECTIVES: ")
			for i := range objectives {
				fmt.Printf("  - %s\n", objectives[i].ID)
			}
		}

		// Job objective is required.
		for jobObjective == "" {
			err := readLine("Job Objective: ", &jobObjective)
			if err != nil {
				fmt.Printf(err.Error() + "\n")
				return
			}
		}

		// Job Alternative Objectives is optional.
		if len(jobAltObjectives) == 0 {
			var altObjectivesString string
			err := readLine("Job Alternative Objectives [optional, space-separated]: ", &altObjectivesString)
			if err != nil {
				fmt.Printf(err.Error() + "\n")
				return
			}
			jobAltObjectives = strings.Fields(altObjectivesString)
		}

		// Find all applicable models.
		models, err := context.GetModules(model.ModuleModel, "", model.ModuleActive, "", dataset.SchemaIn, dataset.SchemaOut)
		if err != nil {
			fmt.Println("Error: " + err.Error())
			return
		}
		if len(models) == 0 {
			fmt.Println("Error: No models applicable to the given dataset.")
			return
		}

		if len(jobModels) == 0 {
			fmt.Println("APPLICABLE MODELS: ")
			for i := range models {
				fmt.Printf("  - %s\n", models[i].ID)
			}
		}

		// Job Models is required.
		for len(jobModels) == 0 {
			var modelsString string
			err := readLine("Job Models [space-separated]: ", &modelsString)
			if err != nil {
				fmt.Printf(err.Error() + "\n")
				return
			}
			jobModels = strings.Fields(modelsString)
		}

		//If an asterisk was specified, replace with all applicable models.
		if len(jobModels) == 1 && jobModels[0] == "*" {
			jobModels = []string{}
			for i := range models {
				jobModels = append(jobModels, models[i].ID)
			}
		}

		// TODO: Refine this. Enable us to detect when a flag wasn't set.

		result, err := context.CreateJob(jobDataset, jobObjective, jobModels, jobAltObjectives, jobAcceptNewModels, jobMaxTasks)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Printf("SUCCESS: Job \"%s\" created.\n", result)

	},
}

func init() {
	createCmd.AddCommand(createJobCmd)

	createJobCmd.Flags().StringVar(&jobDataset, "dataset", "", "Job dataset.")
	createJobCmd.Flags().StringVar(&jobObjective, "objective", "", "Job objective.")
	createJobCmd.Flags().StringArrayVar(&jobModels, "models", []string{}, "Models to apply to the job. "+
		"Asterisk (*) denotes all applicable models.")
	createJobCmd.Flags().StringArrayVar(&jobAltObjectives, "alt-objectives", []string{}, "Job alternative objectives.")
	createJobCmd.Flags().BoolVar(&jobAcceptNewModels, "accept-new-models", false, "Set to indicate that new models "+
		"applicable to the job will also be added.")
	createJobCmd.Flags().Uint64Var(&jobMaxTasks, "max-tasks", model.DefaultMaxTasks, "Maximum number of tasks to spawn from this job.")

}
