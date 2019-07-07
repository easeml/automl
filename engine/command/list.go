package command

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Lists items of a given type.",
	Long:  ``,
}

func init() {
	rootCmd.AddCommand(listCmd)

	//loginCmd.PersistentFlags().BoolVarP(&saveAPIKey, "save", "s", false, "Write the resulting API key to the config file.")

	viper.BindPFlags(listCmd.PersistentFlags())

}
