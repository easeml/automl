package command

import (
	"github.com/ds3lab/easeml/client/go/client"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var listDatasetStatus, listDatasetSource, listDatasetSchemaIn, listDatasetSchemaOut string

var listDatasetsCmd = &cobra.Command{
	Use:   "datasets",
	Short: "Lists datasets.",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

		if apiKey == "" {
			apiKey = viper.GetString("api-key")
		}
		serverAddress := viper.GetString("server-address")
		context := client.Context{ServerAddress: serverAddress, UserCredentials: client.APIKeyCredentials{APIKey: apiKey}}

		var schemaStringIn string
		if listDatasetSchemaIn != "" {
			var err error
			schemaStringIn, err = loadStream(listDatasetSchemaIn)
			if err != nil {
				fmt.Println("Error: " + err.Error())
				return
			}
		}

		var schemaStringOut string
		if listDatasetSchemaOut != "" {
			var err error
			schemaStringOut, err = loadStream(listDatasetSchemaOut)
			if err != nil {
				fmt.Println("Error: " + err.Error())
				return
			}
		}

		result, err := context.GetDatasets(listDatasetStatus, listDatasetSource, schemaStringIn, schemaStringOut)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Printf("Number of results: %d\n\n", len(result))

		if len(result) > 0 {
			w := tabwriter.NewWriter(os.Stdout, 0, 8, 3, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tSTATUS\tSOURCE\tSOURCE ADDRESS\tCREATION TIME")

			for _, r := range result {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", r.ID, r.Name, r.Status, r.Source, r.SourceAddress, r.CreationTime.String())
			}

			w.Flush()

		}

	},
}

func init() {
	listCmd.AddCommand(listDatasetsCmd)

	listDatasetsCmd.Flags().StringVar(&listDatasetStatus, "status", "", "Filter datasets by status.")
	listDatasetsCmd.Flags().StringVar(&listDatasetSource, "source", "", "Filter datasets by source.")
	listDatasetsCmd.Flags().StringVar(&listDatasetSchemaIn, "schema-in", "", "Filter datasets by input schema. "+
		"Can be a path to a schema file or \"-\" in order to read the schema from stdin.")
	listDatasetsCmd.Flags().StringVar(&listDatasetSchemaOut, "schema-out", "", "Filter datasets by output schema. "+
		"Can be a path to a schema file or \"-\" in order to read the schema from stdin.")

}
