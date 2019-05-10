package command

import (
	"github.com/ds3lab/easeml/client/go/client"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var listModuleType, listModuleUser, listModuleStatus, listModuleSource, listModuleSchemaIn, listModuleSchemaOut string

var listModulesCmd = &cobra.Command{
	Use:   "modules",
	Short: "Lists modules.",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

		if apiKey == "" {
			apiKey = viper.GetString("api-key")
		}
		serverAddress := viper.GetString("server-address")
		context := client.Context{ServerAddress: serverAddress, UserCredentials: client.APIKeyCredentials{APIKey: apiKey}}

		var schemaStringIn string
		if listModuleSchemaIn != "" {
			var err error
			schemaStringIn, err = loadStream(listModuleSchemaIn)
			if err != nil {
				fmt.Println("Error: " + err.Error())
				return
			}
		}

		var schemaStringOut string
		if listModuleSchemaOut != "" {
			var err error
			schemaStringOut, err = loadStream(listModuleSchemaOut)
			if err != nil {
				fmt.Println("Error: " + err.Error())
				return
			}
		}

		result, err := context.GetModules(listModuleType, listModuleUser, listModuleStatus, listModuleSource, schemaStringIn, schemaStringOut)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Printf("Number of results: %d\n\n", len(result))

		if len(result) > 0 {
			w := tabwriter.NewWriter(os.Stdout, 0, 8, 3, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tTYPE\tSTATUS\tSOURCE\tSOURCE ADDRESS\tCREATION TIME")

			for _, r := range result {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", r.ID, r.Name, r.Type, r.Status, r.Source, r.SourceAddress, r.CreationTime.String())
			}

			w.Flush()

		}

	},
}

func init() {
	listCmd.AddCommand(listModulesCmd)

	listModulesCmd.Flags().StringVar(&listModuleType, "type", "", "Filter modules by type.")
	listModulesCmd.Flags().StringVar(&listModuleUser, "user", "", "Filter modules by user.")
	listModulesCmd.Flags().StringVar(&listModuleStatus, "status", "", "Filter modules by status.")
	listModulesCmd.Flags().StringVar(&listModuleSource, "source", "", "Filter modules by source.")
	listModulesCmd.Flags().StringVar(&listModuleSchemaIn, "schema-in", "", "Filter modules by input schema. "+
		"Can be a path to a schema file or \"-\" in order to read the schema from stdin.")
	listModulesCmd.Flags().StringVar(&listModuleSchemaOut, "schema-out", "", "Filter modules by output schema. "+
		"Can be a path to a schema file or \"-\" in order to read the schema from stdin.")

}
