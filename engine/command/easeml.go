// Copyright © 2018 DS3LAB
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package command

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var homeDir string
var defaultWorkingDir string
var workingDir string

const defaultConfigFileName = "config"

var apiKey string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "easeml",
	Short: "Ease.ml is an automated machine learning service",
	Long: `Ease.ml is an automated machine learning service that hosts datasets and ML models.
It enables users to define model selection jobs given a dataset and objective function. The system
then performs autmated model selection and hyperparameter tuning.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// GetRootCommand returns the root command.
func GetRootCommand() *cobra.Command {
	return rootCmd
}

func init() {

	// Find home directory.
	var err error
	homeDir, err = homedir.Dir()
	if err != nil {
		panic(err)
	}
	defaultWorkingDir = filepath.Join(homeDir, ".easeml")

	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Location of the config file.")

	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "API key of the user.")

	viper.BindPFlag("api-key", rootCmd.PersistentFlags().Lookup("api-key"))

	// Default working directory is called ".easeml" and placed in the user's home directory.
	rootCmd.PersistentFlags().StringVar(&workingDir, "working-dir", defaultWorkingDir,
		"Path to the working directory that stores all ease.ml files.")
	viper.BindPFlag("working-dir", rootCmd.PersistentFlags().Lookup("working-dir"))

	rootCmd.Long = getEasemlSign() + "\n" + rootCmd.Long
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {

	setConfigInfo(viper.GetViper())

	// Viper's environment variable for 12 factor app style configuration.
	// The equivalence between flags and environmental variables is as follows:
	// [flag] --database-address ADDRESS is equivalent to [env] export EASEML_DATABASE_ADDRESS="ADDRESS" (aside from precedence)
	viper.SetEnvPrefix("EASEML")
	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Error while reading config file:", err)
	}

}

func setConfigInfo(v *viper.Viper) {

	if cfgFile != "" {
		// Use config file from the flag.
		v.SetConfigFile(cfgFile)
	} else {

		// Search config in home directory with name ".easeml" (without extension).
		v.AddConfigPath(workingDir)
		v.SetConfigName(defaultConfigFileName)
	}

}

// updateConfigAndWrite takes a map of keys and values and updates the config file with them.
// If no config file is found, a new one is created.
func updateConfigAndWrite(values map[string]interface{}) (err error) {
	v := viper.New()
	setConfigInfo(v)
	v.ReadInConfig()
	for k := range values {
		v.Set(k, values[k])
	}
	if v.ConfigFileUsed() == "" {
		v.SetConfigFile(filepath.Join(workingDir, defaultConfigFileName+".yaml"))
	}
	err = v.WriteConfig()

	if err == nil {
		fmt.Printf("Updated config file: %s\n", v.ConfigFileUsed())
	}

	return
}

func getEasemlSign() string {
	var easemlSign string
	easemlSign += "                                            __ " + "\n"
	easemlSign += "    ___   ____ _ _____ ___      ____ ___   / / " + "\n"
	easemlSign += "   / _ \\ / __ `// ___// _ \\    / __ `__ \\ / /  " + "\n"
	easemlSign += "  /  __// /_/ /(__  )/  __/_  / / / / / // /   " + "\n"
	easemlSign += "  \\___/ \\__,_//____/ \\___/(_)/_/ /_/ /_//_/    " + "\n"
	return easemlSign
}

func printEasemlSign() {
	easemlSign := getEasemlSign()
	fmt.Println(easemlSign)
}
