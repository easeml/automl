// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
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
	"log"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/ds3lab/easeml/process"
	"github.com/ds3lab/easeml/process/controller"
	"github.com/ds3lab/easeml/process/scheduler"
	"github.com/ds3lab/easeml/process/worker"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var startLogin, openInBrowser bool

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:       "start [service,]",
	Short:     "Starts an ease.ml service.",
	Long:      ``,
	Args:      cobra.OnlyValidArgs,
	ValidArgs: []string{"controller", "scheduler", "worker"},
	Run: func(cmd *cobra.Command, args []string) {

		printEasemlSign()

		keepaliveMilliseconds := viper.GetInt("keepalive-period")
		listenerMilliseconds := viper.GetInt("listener-period")
		//optimizerID := viper.GetString("optimizer")

		processContext := process.Context{
			DatabaseAddress: viper.GetString("database-address"),
			DatabaseName:    viper.GetString("database-name"),
			ServerAddress:   viper.GetString("server-address"),
			WorkingDir:      viper.GetString("working-dir"),
			KeepAlivePeriod: time.Duration(keepaliveMilliseconds) * time.Millisecond,
			ListenerPeriod:  time.Duration(listenerMilliseconds) * time.Millisecond,
			OptimizerID:     "user1/opt-rand-search",
			RootAPIKey:      make(chan string, 1),
		}

		var wg sync.WaitGroup

		if len(args) > 0 {
			for i := range args {
				switch args[i] {
				case "controller":
					wg.Add(1)
					go func() {
						defer wg.Done()
						controller.Start(processContext)
					}()
				case "scheduler":
					wg.Add(1)
					go func() {
						defer wg.Done()
						scheduler.Start(processContext)
					}()
				case "worker":
					wg.Add(1)
					go func() {
						defer wg.Done()
						worker.Start(processContext)
					}()
				}
			}
		} else {
			wg.Add(3)
			go func() {
				defer wg.Done()
				controller.Start(processContext)
			}()
			go func() {
				defer wg.Done()
				scheduler.Start(processContext)
			}()
			go func() {
				defer wg.Done()
				worker.Start(processContext)
			}()
		}

		// Receive API key from processes and save if that was specified.
		go func() {
			gotAPIKey := false
			browserOpened := false
			for rootAPIKey := range processContext.RootAPIKey {
				if startLogin && gotAPIKey == false {
					err := updateConfigAndWrite(map[string]interface{}{"api-key": rootAPIKey})
					if err != nil {
						panic(err)
					}
					gotAPIKey = true
				}
				if openInBrowser && browserOpened == false {

					/* rawQuery := url.Values{"api-key": []string{rootAPIKey}}.Encode()
					webAPIURL := url.URL{
						Scheme:   "http",
						Host:     processContext.ServerAddress,
						Path:     "#/login",
						RawQuery: rawQuery,
					} */
					webAPIURLString := "http://" + processContext.ServerAddress + "/#/login?api-key=" + rootAPIKey

					openbrowser(webAPIURLString)
					browserOpened = true
				}
			}
		}()

		wg.Wait()
	},
}

func init() {
	rootCmd.AddCommand(startCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// startCmd.PersistentFlags().String("foo", "", "A help for foo")
	startCmd.PersistentFlags().String("database-address", "localhost",
		"Connection string for a MongoDB database server formatted as host[:port].")
	startCmd.PersistentFlags().String("database-name", "easeml",
		"Name of the database that hosts all of the ease.ml state.")

	startCmd.PersistentFlags().String("server-address", "localhost:8080",
		"Host address and port of the ease.ml controller process.")

	startCmd.PersistentFlags().Uint("keepalive-period", 1000,
		"Duration in miliseconds between two keepalive messages.")
	startCmd.PersistentFlags().Uint("listener-period", 250,
		"Duration in miliseconds between two database listener queries.")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// startCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	startCmd.Flags().BoolVar(&startLogin, "login", false, "Add root API key to the config so that it can be used for CLI access.")
	startCmd.Flags().BoolVar(&openInBrowser, "browser", false, "Open browser window with the web UI.")

	// Bind with viper config.
	viper.BindPFlags(startCmd.PersistentFlags())
	//viper.BindPFlag("database-address", startCmd.PersistentFlags().Lookup("database-address"))
	//viper.BindPFlag("database-name", startCmd.PersistentFlags().Lookup("database-name"))
	//viper.BindPFlag("working-dir", startCmd.PersistentFlags().Lookup("working-dir"))
}

func openbrowser(url string) {
	var err error

	fmt.Println(url)

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}

}
