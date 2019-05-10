package command

import (
	"github.com/ds3lab/easeml/engine/easeml/modules"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var runTrainData, runTrainConfig, runTrainOutput string

var runTrainCmd = &cobra.Command{
	Use:   "train [image]",
	Short: "Runs a train command on a model given its docker image.",
	Long:  ``,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		modelImageName := args[0]

		// Ensure all input parameters point to existing files and folders.
		if _, err := os.Stat(runTrainData); os.IsNotExist(err) {
			fmt.Printf("Data path \"%s\" doesn't exist.\n", runTrainData)
			return
		}
		if _, err := os.Stat(runTrainConfig); os.IsNotExist(err) {
			fmt.Printf("Config path \"%s\" doesn't exist.\n", runTrainConfig)
			return
		}
		if _, err := os.Stat(runTrainOutput); os.IsNotExist(err) {
			fmt.Printf("Output path \"%s\" doesn't exist.\n", runTrainOutput)
			return
		}

		command := []string{
			"train",
			"--data", modules.MntPrefix + runTrainData,
			"--conf", modules.MntPrefix + runTrainConfig,
			"--output", modules.MntPrefix + runTrainOutput,
		}
		outReader, err := modules.RunContainerAndCollectOutput(modelImageName, nil, command)
		if err != nil {
			fmt.Println("Error while running the container: ")
			fmt.Print(err)
		}
		defer outReader.Close()

		// Read the output reader and write it to stdout.
		trainLogData, err := ioutil.ReadAll(outReader)
		fmt.Print(string(trainLogData))

	},
}

func init() {
	runCmd.AddCommand(runTrainCmd)

	//loginCmd.PersistentFlags().BoolVarP(&saveAPIKey, "save", "s", false, "Write the resulting API key to the config file.")

	runTrainCmd.Flags().StringVarP(&runTrainData, "data", "d", "", "Directory containing the training data.")
	runTrainCmd.Flags().StringVarP(&runTrainConfig, "conf", "c", "", "Model configuration.")
	runTrainCmd.Flags().StringVarP(&runTrainOutput, "output", "o", "", "Directory where the model will output its parameters.")

	viper.BindPFlags(runTrainCmd.PersistentFlags())

}
