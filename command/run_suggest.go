package command

import (
	"github.com/ds3lab/easeml/modules"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var runSuggestSpace, runSuggestHistory string
var runSuggestNumTasks int

var runSuggestCmd = &cobra.Command{
	Use:   "suggest [image]",
	Short: "Runs a suggest command on an optimizer given its docker image.",
	Long:  ``,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		modelImageName := args[0]

		// Ensure all input parameters point to existing files and folders.
		if _, err := os.Stat(runSuggestSpace); os.IsNotExist(err) {
			fmt.Printf("Config space path \"%s\" doesn't exist.\n", runSuggestSpace)
			return
		}
		if _, err := os.Stat(runSuggestHistory); os.IsNotExist(err) {
			fmt.Printf("History path \"%s\" doesn't exist.\n", runSuggestHistory)
			return
		}

		command := []string{
			"suggest",
			"--space", modules.MntPrefix + runSuggestSpace,
			"--history", modules.MntPrefix + runSuggestHistory,
			"--num-tasks", strconv.Itoa(runSuggestNumTasks),
		}
		outReader, err := modules.RunContainerAndCollectOutput(modelImageName, nil, command)
		if err != nil {
			fmt.Println("Error while running the container: ")
			fmt.Print(err)
		}
		defer outReader.Close()

		// Read the output reader and write it to stdout.
		suggestLogData, err := ioutil.ReadAll(outReader)
		fmt.Print(string(suggestLogData))

	},
}

func init() {
	runCmd.AddCommand(runSuggestCmd)

	//loginCmd.PersistentFlags().BoolVarP(&saveAPIKey, "save", "s", false, "Write the resulting API key to the config file.")

	runSuggestCmd.Flags().StringVarP(&runSuggestSpace, "space", "s", "", "Directory containing the search space. "+
		"Each JSON file represents one branch of the search space. All of them will be concatenated using a .choice element.")
	runSuggestCmd.Flags().StringVarP(&runSuggestHistory, "history", "i", "", "Directory containing the history.")
	runSuggestCmd.Flags().IntVarP(&runSuggestNumTasks, "num-tasks", "n", 1, "Number of new tasks the optimizers should suggest.")

	viper.BindPFlags(runSuggestCmd.PersistentFlags())

}
