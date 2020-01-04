package command

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ds3lab/easeml/engine/modules"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var runEvalActual, runEvalPredicted string

var runEvalCmd = &cobra.Command{
	Use:   "eval [image]",
	Short: "Runs the eval command on an objective given its docker image.",
	Long:  ``,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		modelImageName := args[0]

		// Ensure all input parameters point to existing files and folders.
		if _, err := os.Stat(runEvalActual); os.IsNotExist(err) {
			fmt.Printf("Actual data path \"%s\" doesn't exist.\n", runEvalActual)
			return
		}
		if _, err := os.Stat(runEvalPredicted); os.IsNotExist(err) {
			fmt.Printf("Predictions data path \"%s\" doesn't exist.\n", runEvalPredicted)
			return
		}

		command := []string{
			"eval",
			"--actual", modules.MntPrefix + runEvalActual,
			"--predicted", modules.MntPrefix + runEvalPredicted,
		}
		outReader, err := modules.RunContainerAndCollectOutput(modelImageName, nil, command, nil)
		if err != nil {
			fmt.Println("Error while running the container: ")
			fmt.Print(err)
		}
		defer outReader.Close()

		// Read the output reader and write it to stdout.
		evalLogData, err := ioutil.ReadAll(outReader)
		fmt.Print(string(evalLogData))

	},
}

func init() {
	runCmd.AddCommand(runEvalCmd)

	//loginCmd.PersistentFlags().BoolVarP(&saveAPIKey, "save", "s", false, "Write the resulting API key to the config file.")

	runEvalCmd.Flags().StringVarP(&runEvalActual, "actual", "a", "", "Directory containing the actual validation data.")
	runEvalCmd.Flags().StringVarP(&runEvalPredicted, "predicted", "p", "", "Directory containint the predictions.")

	viper.BindPFlags(runEvalCmd.PersistentFlags())

}
