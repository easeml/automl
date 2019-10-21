package command

import (
	"fmt"
	"os"
	"path"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var globalInstall, forceMode bool

var setupInstallCmd = &cobra.Command{
	Use:       "install [component,]",
	Short:     "Executes an installation script for one or more components such as docker or mongo.",
	Long:      ``,
	Args:      cobra.OnlyValidArgs,
	ValidArgs: []string{"docker", "mongo"},
	Run: func(cmd *cobra.Command, args []string) {

		for i := range args {
			// The script path will either be docker/install or mongo/install.
			scriptName := path.Join(args[i], "install")

			// Add script arguments if needed.
			scriptArgs := []string{}
			if globalInstall {
				// In global mode, the script knows where to install the target dependency.
				scriptArgs = append(scriptArgs, "-g")
			} else {
				// In non-global mode, we install the target dependency in the working directory.
				scriptArgs = append(scriptArgs, "-c")
				scriptArgs = append(scriptArgs, path.Join(workingDir, "opt", args[i]))
			}
			if forceMode {
				scriptArgs = append(scriptArgs, "-f")
			}

			// Run the script and plug in standard output and standard error.
			_, err := runEmbeddedScriptAsync(scriptName, scriptArgs, os.Stdout, os.Stderr)
			if err != nil {
				fmt.Printf(err.Error() + "\n")
				return
			}
		}

	},
}

func init() {

	// Only include this command if the installation scripts are available.
	if scriptsAvailable {
		setupCmd.AddCommand(setupInstallCmd)
	}

	setupInstallCmd.Flags().BoolVarP(&globalInstall, "global", "g", false, "Perform system-wide installation. Must be run as root/admin user.")

	setupInstallCmd.Flags().BoolVarP(&forceMode, "force", "f", false, "Force mode. Never prompt the user for confirmation.")

	viper.BindPFlags(setupInstallCmd.PersistentFlags())

}
