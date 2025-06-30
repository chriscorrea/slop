package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "slop",
	Short: "A CLI tool for interacting with LLMs",
	Long:  "Slop brings large language models to your command line, enabling pipeline-native interactions and streamlined text processing.",
}

// called by main.main()
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// define your flags and config settings
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.slop.toml)")

	// local flags will only run when the action is called directly
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// find home dir
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// search config in home directory with name ".slop" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("toml")
		viper.SetConfigName(".slop")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// if a config file is found, read it in!
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
