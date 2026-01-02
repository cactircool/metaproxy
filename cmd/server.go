/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/cactircool/metaproxy/server"
	"github.com/cactircool/metaproxy/util"
	"github.com/spf13/cobra"
)

type ServerFlags struct {
	Verbose *bool
}

var serverFlags = ServerFlags{}

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server [flags] [...CONFIG_FILES]",
	Short: "Run a metaproxy server",
	Long: `Starts a metaproxy server that accepts incoming client connections
and forwards traffic according to its configuration.

This command is typically used on a host that will act as the
proxy endpoint for one or more clients.

Examples:
  metaproxy server mp.cfg
  metaproxy server --verbose mp.cfg`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		startServer(args)
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serverCmd.PersistentFlags().String("foo", "", "A help for foo")

	serverFlags.Verbose = serverCmd.Flags().BoolP("verbose", "v", false, "enable verbose logging for servers")
}

func startServer(args []string) {
	if *serverFlags.Verbose {
		util.SetVerbose(true)
	}

	for _, configPath := range args {
		file, err := os.Open(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s:\n\tfailed to open config file: %v\n", configPath, err)
			os.Exit(1)
		}
		defer file.Close()

		if err := server.ConfigStart(file); err != nil {
			fmt.Fprintf(os.Stderr, "%s:\n\tfailed to parse config and start server: %v\n", configPath, err)
			os.Exit(1)
		}
	}

	util.Logln(os.Stdout, "All servers up and running...")
	select {}
}
