package command

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ds3lab/easeml/engine/modules"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var runPredictData, runPredictMemory, runPredictOutput string
var runPredictGpuDevices []string

const defaultMemory = "/memory"

var runPredictCmd = &cobra.Command{
	Use:   "predict [image]",
	Short: "Runs a predict command on a model given its docker image.",
	Long:  ``,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		modelImageName := args[0]

		// Ensure all input parameters point to existing files and folders.
		if _, err := os.Stat(runPredictData); os.IsNotExist(err) {
			fmt.Printf("Data path \"%s\" doesn't exist.\n", runPredictData)
			return
		}
		if runPredictMemory == "" {
			// We allow the memory to be omitted.
			// Then we assume the model is trained and has a local /memory directory.
			runPredictMemory = defaultMemory
		} else if _, err := os.Stat(runPredictMemory); os.IsNotExist(err) {
			fmt.Printf("Memory path \"%s\" doesn't exist.\n", runPredictMemory)
			return
		}
		if _, err := os.Stat(runPredictOutput); os.IsNotExist(err) {
			fmt.Printf("Output path \"%s\" doesn't exist.\n", runPredictOutput)
			return
		}

		// If the image is a tar file we load it.
		fileStat, err := os.Stat(modelImageName)
		if err == nil {
			if fileStat.IsDir() == false {
				modelImagePath := modelImageName
				modelImageName, err = modules.LoadImage(modelImagePath)
				if err != nil {
					fmt.Printf("Error while loading image from \"%s\":\n", modelImagePath)
					fmt.Println(err)
					return
				}
			}
		}

		command := []string{
			"predict",
			"--data", modules.MntPrefix + runPredictData,
			"--memory", modules.MntPrefix + runPredictMemory,
			"--output", modules.MntPrefix + runPredictOutput,
		}
		outReader, err := modules.RunContainerAndCollectOutput(modelImageName, nil, command, runPredictGpuDevices)
		if err != nil {
			fmt.Println("Error while running the container: ")
			fmt.Print(err)
		}
		defer outReader.Close()

		// Read the output reader and write it to stdout.
		predictLogData, err := ioutil.ReadAll(outReader)
		fmt.Print(string(predictLogData))

	},
}

func init() {
	runCmd.AddCommand(runPredictCmd)

	//loginCmd.PersistentFlags().BoolVarP(&saveAPIKey, "save", "s", false, "Write the resulting API key to the config file.")

	runPredictCmd.Flags().StringVarP(&runPredictData, "data", "d", "", "Directory containing the input data.")
	runPredictCmd.Flags().StringVarP(&runPredictMemory, "memory", "m", "", "Model memory.")
	runPredictCmd.Flags().StringVarP(&runPredictOutput, "output", "o", "", "Directory where the model will output its predictions.")
	runPredictCmd.Flags().StringSliceVar(&runPredictGpuDevices, "gpu", []string{}, "List of integer GPU device identifiers "+
		"(zero based) that specified which GPU devices to make available to each module executed by a worker "+
		"process. If -1 is specified, then all GPU devices are made available. If empty, then only CPU is used. "+
		"For example --gpu 0,2 means that GPU0 and GPU2 will be made available to the module. "+
		"NVidia Docker runtime needs to be installed to use this feature (https://github.com/NVIDIA/nvidia-docker).")

	viper.BindPFlags(runPredictCmd.PersistentFlags())

}
