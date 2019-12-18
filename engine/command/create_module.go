package command

import (
	"fmt"
	"strings"

	client "github.com/ds3lab/easeml/client/go/easemlclient"
	"github.com/ds3lab/easeml/client/go/easemlclient/types"
	"github.com/ds3lab/easeml/engine/modules"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var moduleID, moduleType, moduleLabel, moduleName, moduleDescription, moduleSchema, moduleSource, moduleSourceAddress string

var createModuleCmd = &cobra.Command{
	Use:   "module",
	Short: "Creates a new module.",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

		if apiKey == "" {
			apiKey = viper.GetString("api-key")
		}
		serverAddress := viper.GetString("server-address")
		context := client.Context{ServerAddress: serverAddress, UserCredentials: client.APIKeyCredentials{APIKey: apiKey}}

		// Module type is required.
		for client.ModuleTypeValid(moduleType) == false {
			prompt := fmt.Sprintf("Module Type [choices - %s]: ", strings.Join(client.ValidModuleTypes, ", "))
			err := readLine(prompt, &moduleType)
			if err != nil {
				fmt.Printf(err.Error() + "\n")
				return
			}
		}

		// Module Source is required.
		for client.ModuleSourceValid(moduleSource) == false {
			prompt := fmt.Sprintf("Module Source [choices - %s]: ", strings.Join(client.ValidModuleSources, ", "))
			err := readLine(prompt, &moduleSource)
			if err != nil {
				fmt.Printf(err.Error() + "\n")
				return
			}
		}

		// Module Source is required.
		for moduleSourceAddress == "" {
			err := readLine("Module Source Address: ", &moduleSourceAddress)
			if err != nil {
				fmt.Printf(err.Error() + "\n")
				return
			}
		}

		// If the source is upload, the data is available to us and we
		// will try to infer its id, name and description.
		var defaultID, defaultName, defaultDescription string
		if moduleSource == types.ModuleUpload {
			var err error
			defaultID, defaultName, defaultDescription, _, _, _, err =
				modules.InferModuleProperties(moduleSourceAddress)

			if err != nil {
				fmt.Printf(err.Error() + "\n")
				return
			}
		}

		// Module ID is required. We can use a default if available.
		for moduleID == "" {
			prompt := "Module ID"
			if defaultID != "" {
				prompt += fmt.Sprintf(" [default: %s]", defaultID)
			}
			prompt += ": "
			err := readLine(prompt, &moduleID)
			if err != nil {
				fmt.Printf(err.Error() + "\n")
				return
			}
			if moduleID == "" {
				moduleID = defaultID
			}
		}

		// If the module id is given in the short format, we reformat it if possible.
		ids := strings.Split(moduleID, "/")
		if len(ids) == 1 {
			user, err := context.GetUserByID("")
			if err != nil {
				fmt.Println("Error: " + err.Error())
			}
			moduleID = fmt.Sprintf("%s/%s", user.ID, moduleID)
		} else if len(ids) != 2 {
			fmt.Println("Error: The id must be of the format user-id/module-id")
			return
		}

		// Check if the Module ID is taken. Maybe we need to abort, or just skip to the upload part.
		if module, _ := context.GetModuleByID(moduleID); module == nil {

			// Module Name is optional. We can use a default if available.
			if moduleName == "" {
				prompt := "Module Name [optional"
				if defaultName != "" {
					prompt += fmt.Sprintf(", default: %s", defaultName)
				}
				prompt += "]: "

				err := readLine(prompt, &moduleName)
				if err != nil {
					fmt.Printf(err.Error() + "\n")
					return
				}
				if moduleName == "" {
					moduleName = defaultName
				}
			}

			// Module description is optional.
			var descriptionString = defaultDescription
			if moduleDescription != "" {
				var err error
				descriptionString, err = loadStream(moduleDescription)
				if err != nil {
					fmt.Println("Error: " + err.Error())
					return
				}
			}

			// Module Label is optional.
			if moduleLabel == "" {
				prompt := "Module Label [optional]:"
				err := readLine(prompt, &moduleLabel)
				if err != nil {
					fmt.Printf(err.Error() + "\n")
					return
				}
			}

			_, err := context.CreateModule(moduleID, moduleType, moduleLabel, moduleName, descriptionString, moduleSource, moduleSourceAddress)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			// TODO: Poll the module status until it becomes "ready", to enable the user to have feedback about the process
			fmt.Printf("SUCCESS: Module \"%s\" created.\n", moduleID)

		} else if module.Source != types.ModuleUpload || module.Status != types.ModuleCreated {
			fmt.Printf("Error: Module \"%s\" already exists.\n", moduleID)
			return
		}

		// Perform upload if the source is upload.
		if moduleSource == types.ModuleUpload {
			err := context.UploadModule(moduleID, moduleSourceAddress)
			if err != nil {
				fmt.Println("Upload Error: " + err.Error())
				return
			}
			fmt.Printf("SUCCESS: Module \"%s\" uploaded.\n", moduleID)
		}

	},
}

func init() {
	createCmd.AddCommand(createModuleCmd)

	createModuleCmd.Flags().StringVar(&moduleID, "id", "", "Module ID.")
	createModuleCmd.Flags().StringVar(&moduleType, "type", "", "Module type.")
	createModuleCmd.Flags().StringVar(&moduleLabel, "label", "", "Module label.")
	createModuleCmd.Flags().StringVar(&moduleName, "name", "", "Module full name.")
	createModuleCmd.Flags().StringVar(&moduleDescription, "description", "", "Module description. "+
		"Can be a path to a text file or \"-\" in order to read the description from stdin.")
	createModuleCmd.Flags().StringVar(&moduleSource, "source", "", "Module source.")
	createModuleCmd.Flags().StringVar(&moduleSourceAddress, "source-address", "", "Module source address.")

}
