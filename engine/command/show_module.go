package command

import (
	"github.com/ds3lab/easeml/client/go/easemlclient"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var showModuleCmd = &cobra.Command{
	Use:   "module id",
	Short: "Shows module given its id.",
	Long:  ``,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		if apiKey == "" {
			apiKey = viper.GetString("api-key")
		}
		serverAddress := viper.GetString("server-address")
		context := client.Context{ServerAddress: serverAddress, UserCredentials: client.APIKeyCredentials{APIKey: apiKey}}

		id := ""
		if len(args) > 0 {
			id = args[0]
		}

		result, err := context.GetModuleByID(id)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 8, 3, ' ', 0)

		fmt.Fprintf(w, "ID:\t%s\n", result.ID)
		fmt.Fprintf(w, "TYPE:\t%s\n", result.Type)
		fmt.Fprintf(w, "NAME:\t%s\n", result.Name)
		fmt.Fprintf(w, "STATUS:\t%s\n", result.Status)
		if result.StatusMessage != "" {
			fmt.Fprintf(w, "STATUS MESSAGE:\t%s\n", result.StatusMessage)
		}
		fmt.Fprintf(w, "SOURCE:\t%s\n", result.Source)
		fmt.Fprintf(w, "SOURCE ADDRESS:\t%s\n", result.SourceAddress)
		fmt.Fprintf(w, "CREATION TIME:\t%s\n", result.CreationTime)
		w.Flush()

		if result.Description != "" {
			fmt.Printf("DESCRIPTION:\n\n%s\n", result.Description)
		}

		if result.SchemaIn != "" {
			ymlSchemaIn, err := yaml.JSONToYAML([]byte(result.SchemaIn))
			if err != nil {
				fmt.Printf("INPUT SCHEMA: (format error: %s)\n%s\n", err, result.SchemaIn)
			} else {
				fmt.Printf("INPUT SCHEMA:\n\n%s\n", string(ymlSchemaIn))
			}
		}

		if result.SchemaOut != "" {
			ymlSchemaOut, err := yaml.JSONToYAML([]byte(result.SchemaOut))
			if err != nil {
				fmt.Printf("OUTPUT SCHEMA: (format error: %s)\n%s\n", err, result.SchemaOut)
			} else {
				fmt.Printf("OUTPUT SCHEMA:\n\n%s\n", string(ymlSchemaOut))
			}
		}

	},
}

func init() {
	showCmd.AddCommand(showModuleCmd)
}
