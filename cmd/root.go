/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "metaproxy",
	Short: "A flexible proxy server and client for protocol-agnostic traffic forwarding",
	Long: `metaproxy is a client/server proxy system that forwards arbitrary
network protocols through configurable endpoints.

It allows clients to connect to a remote metaproxy server and
dynamically route traffic based on protocol, location, or deployment
needs, without requiring protocol-specific support.

Common commands:
  metaproxy connect   Connect to a remote metaproxy server
  metaproxy server    Run a metaproxy server instance

Use "metaproxy <command> --help" for more information about a command.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.metaproxy.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
