package command

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Runs various setup procedures that are needed to run easeml on a given system.",
	Long:  ``,
}

func init() {
	rootCmd.AddCommand(setupCmd)

	//loginCmd.PersistentFlags().BoolVarP(&saveAPIKey, "save", "s", false, "Write the resulting API key to the config file.")

	viper.BindPFlags(setupCmd.PersistentFlags())

}
