package command

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ds3lab/easeml/engine/modules"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var runTrainData, runTrainConfig, runTrainOutput string
var runTrainGpuDevices []string

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
		outReader, err := modules.RunContainerAndCollectOutput(modelImageName, nil, command, runTrainGpuDevices)
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
	runTrainCmd.Flags().StringSliceVar(&runTrainGpuDevices, "gpu", []string{}, "List of integer GPU device identifiers "+
		"(zero based) that specified which GPU devices to make available to each module executed by a worker "+
		"process. If -1 is specified, then all GPU devices are made available. If empty, then only CPU is used. "+
		"For example --gpu 0,2 means that GPU0 and GPU2 will be made available to the module. "+
		"NVidia Docker runtime needs to be installed to use this feature (https://github.com/NVIDIA/nvidia-docker).")

	viper.BindPFlags(runTrainCmd.PersistentFlags())

}
