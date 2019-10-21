package command

import (
	client "github.com/ds3lab/easeml/client/go/easemlclient"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var pushDatasetID, pushDatasetName, pushDatasetDescription string

var pushDatasetCmd = &cobra.Command{
	Use:   "dataset",
	Short: "Pushes a dataset to the ease.ml service.",
	Long:  ``,
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {

		if apiKey == "" {
			apiKey = viper.GetString("api-key")
		}
		serverAddress := viper.GetString("server-address")
		context := client.Context{ServerAddress: serverAddress, UserCredentials: client.APIKeyCredentials{APIKey: apiKey}}

		// Dataset ID is required.
		for datasetID == "" {
			err := readLine("Dataset ID: ", &datasetID)
			if err != nil {
				fmt.Printf(err.Error() + "\n")
				return
			}
		}

		// Dataset Name is optional.
		if datasetName == "" {
			err := readLine("Dataset Name [optional]: ", &datasetName)
			if err != nil {
				fmt.Printf(err.Error() + "\n")
				return
			}
		}

		// Dataset schema is required.
		var descriptionString string
		if datasetDescription != "" {
			var err error
			descriptionString, err = loadStream(datasetDescription)
			if err != nil {
				fmt.Println("Error: " + err.Error())
				return
			}
		}

		// Dataset Source is required.
		for client.DatasetSourceValid(datasetSource) == false {
			prompt := fmt.Sprintf("Dataset Source [choices - %s]: ", strings.Join(client.ValidDatasetSources, ", "))
			err := readLine(prompt, &datasetSource)
			if err != nil {
				fmt.Printf(err.Error() + "\n")
				return
			}
		}

		// Dataset Source is required for some sources.
		if client.DatasetSourceAddressRequired(datasetSource) && datasetSourceAddress == "" {
			err := readLine("Dataset Source Address: ", &datasetSource)
			if err != nil {
				fmt.Printf(err.Error() + "\n")
				return
			}
		}

		result, err := context.CreateDataset(datasetID, datasetName, descriptionString, datasetSource, datasetSourceAddress)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Printf("SUCCESS: Dataset \"%s\" created.\n", result)

	},
}

func init() {
	pushCmd.AddCommand(pushDatasetCmd)

	pushDatasetCmd.Flags().StringVar(&pushDatasetID, "id", "", "Dataset ID. If omitted, it will be taken "+
		"from the dataset directory or archive name.")
	pushDatasetCmd.Flags().StringVar(&pushDatasetName, "name", "", "Dataset full name. If omitted, it "+
		"will be taken from the first line of the README file in the dataset root.")
	pushDatasetCmd.Flags().StringVar(&pushDatasetDescription, "description", "", "Dataset description. "+
		"Can be a path to a text file or \"-\" in order to read the description from stdin. If omitted, it "+
		"will be taken from the first line of the README file in the dataset root.")

}
