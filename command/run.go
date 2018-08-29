package command

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Runs a command on a model given its docker image.",
	Long:  ``,
}

func init() {
	rootCmd.AddCommand(runCmd)

	//loginCmd.PersistentFlags().BoolVarP(&saveAPIKey, "save", "s", false, "Write the resulting API key to the config file.")

	viper.BindPFlags(runCmd.PersistentFlags())

}
