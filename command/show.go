package command

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Shows an item given its id.",
	Long:  ``,
}

func init() {
	rootCmd.AddCommand(showCmd)

	//loginCmd.PersistentFlags().BoolVarP(&saveAPIKey, "save", "s", false, "Write the resulting API key to the config file.")

	viper.BindPFlags(showCmd.PersistentFlags())

}
