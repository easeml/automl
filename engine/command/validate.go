package command

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validates a given item.",
	Long:  ``,
}

func init() {
	rootCmd.AddCommand(validateCmd)

	//loginCmd.PersistentFlags().BoolVarP(&saveAPIKey, "save", "s", false, "Write the resulting API key to the config file.")

	viper.BindPFlags(validateCmd.PersistentFlags())

}
