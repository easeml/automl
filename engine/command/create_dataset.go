package command

import (
	"errors"
	"fmt"
	"strings"

	client "github.com/ds3lab/easeml/client/go/easemlclient"
	"github.com/ds3lab/easeml/client/go/easemlclient/types"
	"github.com/ds3lab/easeml/engine/storage"

	sch "github.com/ds3lab/easeml/schema/go/easemlschema/schema"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var datasetID, datasetName, datasetDescription, datasetSchema, datasetSource, datasetSourceAddress, datasetSecret string

var createDatasetCmd = &cobra.Command{
	Use:   "dataset",
	Short: "Creates a new dataset.",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

		if apiKey == "" {
			apiKey = viper.GetString("api-key")
		}
		serverAddress := viper.GetString("server-address")
		context := client.Context{ServerAddress: serverAddress, UserCredentials: client.APIKeyCredentials{APIKey: apiKey}}

		// Dataset Source is required.
		for client.DatasetSourceValid(datasetSource) == false {
			prompt := fmt.Sprintf("Dataset Source [choices - %s]: ", strings.Join(client.ValidDatasetSources, ", "))
			err := readLine(prompt, &datasetSource)
			if err != nil {
				fmt.Printf(err.Error() + "\n")
				return
			}
		}

		// Dataset Source Address is required.
		for datasetSourceAddress == "" {
			prompt := fmt.Sprintf("Dataset Source Address: ")
			err := readLine(prompt, &datasetSourceAddress)
			if err != nil {
				fmt.Printf(err.Error() + "\n")
				return
			}
		}

		// If the source is upload, the data is available to us and we
		// will try to infer its id, name and description.
		var defaultID, defaultName, defaultDescription string
		if datasetSource == types.DatasetUpload {
			var err error
			defaultID, defaultName, defaultDescription, err = storage.InferDatasetProperties(datasetSourceAddress)
			if err != nil {
				fmt.Printf(err.Error() + "\n")
				return
			}

			// Check if we can infer the schema.
			var schemaIn, schemaOut *sch.Schema
			schemaIn, schemaOut, err = storage.InferDatasetSchema(datasetSourceAddress)
			if err != nil || schemaIn == nil || schemaOut == nil {
				if err == nil {
					err = errors.New("dataset schema missing")
				}

				fmt.Printf(err.Error() + "\n")
				return
			}
		}

		// Dataset ID is required. We can use a default if available.
		for datasetID == "" {
			prompt := "Dataset ID"
			if defaultID != "" {
				prompt += fmt.Sprintf(" [default: %s]", defaultID)
			}
			prompt += ": "
			err := readLine(prompt, &datasetID)
			if err != nil {
				fmt.Printf(err.Error() + "\n")
				return
			}
			if datasetID == "" {
				datasetID = defaultID
			}
		}

		// If the dataset id is given in the short format, we reformat it if possible.
		ids := strings.Split(datasetID, "/")
		if len(ids) == 1 {
			user, err := context.GetUserByID("")
			if err != nil {
				fmt.Println("Error: " + err.Error())
			}
			datasetID = fmt.Sprintf("%s/%s", user.ID, datasetID)
		} else if len(ids) != 2 {
			fmt.Println("Error: The id must be of the format user-id/dataset-id")
			return
		}

		// Check if the Dataset ID is taken. Maybe we need to abort, or just skip to the upload part.
		if dataset, _ := context.GetDatasetByID(datasetID); dataset == nil {

			// Dataset Name is optional. We can use a default if available.
			if datasetName == "" {
				prompt := "Dataset Name [optional"
				if defaultName != "" {
					prompt += fmt.Sprintf(", default: %s", defaultName)
				}
				prompt += "]: "

				err := readLine(prompt, &datasetName)
				if err != nil {
					fmt.Printf(err.Error() + "\n")
					return
				}
				if datasetName == "" {
					datasetName = defaultName
				}
			}

			// Dataset description is optional. We can use a default if available.
			var descriptionString = defaultDescription
			if datasetDescription != "" {
				var err error
				descriptionString, err = loadStream(datasetDescription)
				if err != nil {
					fmt.Println("Error: " + err.Error())
					return
				}
			}

			_, err := context.CreateDataset(datasetID, datasetName, descriptionString, datasetSource, datasetSourceAddress,datasetSecret)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			fmt.Printf("SUCCESS: Dataset \"%s\" creation requested.\n", datasetID)

		} else if dataset.Source != types.DatasetUpload || dataset.Status != types.DatasetCreated {
			fmt.Printf("Error: Dataset \"%s\" already exists.\n", datasetID)
			return
		}

		// Perform upload if the source is upload.
		if datasetSource == types.DatasetUpload {
			err := context.UploadDataset(datasetID, datasetSourceAddress)
			if err != nil {
				fmt.Println("Upload Error: " + err.Error())
				return
			}
			fmt.Printf("SUCCESS: Dataset \"%s\" uploaded.\n", datasetID)
		}
	},
}

func init() {
	createCmd.AddCommand(createDatasetCmd)

	createDatasetCmd.Flags().StringVar(&datasetID, "id", "", "Dataset ID.")
	createDatasetCmd.Flags().StringVar(&datasetName, "name", "", "Dataset full name.")
	createDatasetCmd.Flags().StringVar(&datasetDescription, "description", "", "Dataset description. "+
		"Can be a path to a text file or \"-\" in order to read the description from stdin.")
	createDatasetCmd.Flags().StringVar(&datasetSource, "source", "", fmt.Sprintf("Dataset source [choices: %s]",strings.Join(client.ValidDatasetSources, ", ")))
	createDatasetCmd.Flags().StringVar(&datasetSourceAddress, "source-address", "", "Dataset source address.")
	createDatasetCmd.Flags().StringVar(&datasetSecret, "dataset-secret", "", "Data-source specific secret, i.e. oauth token.")
}
