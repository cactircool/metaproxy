/*
Copyright © 2025 NAME HERE <c4ctircool@gmail.com>
*/
package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/cactircool/metaproxy/client"
	"github.com/spf13/cobra"
)

type ConnectFlags struct {
	LocalPort *int
	OutputPort *bool
}

var connectFlags = ConnectFlags{}

// connectCmd represents the connect command
var connectCmd = &cobra.Command{
	Use:   "connect [flags] PROTOCOL HOST PORT",
	Short: "Connect to a metaproxy server and start proxying traffic",
	Long: `Connects to a metaproxy server and establishes a proxy tunnel.

You must specify:
  PROTOCOL  The protocol to proxy (e.g. tcp, udp)
  HOST      The hostname or IP address of the metaproxy server
  PORT      The server's entry port

Examples:
  metaproxy connect tcp example.com 25565
  metaproxy connect udp 192.168.1.10 9000`,
	Args: cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		startConnect(args)
	},
}

func init() {
	rootCmd.AddCommand(connectCmd) // TODO: figure this out

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// connectCmd.PersistentFlags().String("foo", "", "A help for foo")

	connectFlags.LocalPort = connectCmd.Flags().IntP("local-port", "lp", 0, "specify the port that 'mp connect' runs on, 0 is wildcard; invalid integer ports are silently ignored (default=0).")
	connectFlags.OutputPort = connectCmd.Flags().BoolP("output-port", "op", false, "with this flag set, the first 32 bits written to stdout will contain the port the client is hosted on, followed by the header, then standard protocol (default=false).")
}

func startConnect(args []string) {
	protocol, host, portStr := args[0], args[1], args[2]

	port, err := strconv.Atoi(portStr)
	if err != nil || port < 0 || port > 65535 {
		fmt.Fprintf(os.Stderr, "invalid port: %s\n", portStr)
		os.Exit(1)
	}

	if err := client.Connect(protocol, host, port, *connectFlags.LocalPort, *connectFlags.OutputPort); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}
