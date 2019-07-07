package command

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Creates an item.",
	Long:  ``,
}

func init() {
	rootCmd.AddCommand(createCmd)

	//loginCmd.PersistentFlags().BoolVarP(&saveAPIKey, "save", "s", false, "Write the resulting API key to the config file.")

	viper.BindPFlags(createCmd.PersistentFlags())

}
