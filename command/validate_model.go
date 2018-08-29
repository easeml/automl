package command

import (
	"github.com/ds3lab/easeml/modules"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var validateModelData, validateModelConfig, validateModelOutput string

var validateModelCmd = &cobra.Command{
	Use:   "model [image]",
	Short: "Runs a model command on a model given its docker image.",
	Long:  ``,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		modelImageName := args[0]

		_, _, _, jsonSchemaIn, jsonSchemaOut, configSpace, err := modules.InferModuleProperties(modelImageName)
		if err != nil {
			fmt.Println("Error while getting data from the container: ")
			fmt.Print(err)
			fmt.Println()
		}

		err = modules.ValidateModel(modelImageName, jsonSchemaIn, jsonSchemaOut, configSpace, true)
		if err != nil {
			fmt.Println("Validation failed: ")
			fmt.Print(err)
			fmt.Println()
		}

		fmt.Println("SUCCESS: Validation completed.")
	},
}

func init() {
	validateCmd.AddCommand(validateModelCmd)

	//loginCmd.PersistentFlags().BoolVarP(&saveAPIKey, "save", "s", false, "Write the resulting API key to the config file.")

	viper.BindPFlags(validateModelCmd.PersistentFlags())

}
