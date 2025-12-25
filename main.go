package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/cactircool/metaproxy/client"
	"github.com/cactircool/metaproxy/server"
	"github.com/cactircool/metaproxy/util"
)

func main() {
	flag.BoolFunc(help("help"))
	flag.BoolFunc(help("h"))
	flag.BoolFunc("verbose", "print out server logs (not that in depth)", func(string)error {
		util.SetVerbose(true)
		return nil
	})
	flag.BoolFunc("v", "print out server logs (not that in depth)", func(string)error {
		util.SetVerbose(true)
		return nil
	})
	flag.Parse()

	option := flag.Arg(0)
	switch option {
	case "connect":
		startConnect()

	case "server":
		startServer()

	default:
		usage()
		os.Exit(1)
	}
}

func startConnect() {
	protocol := flag.Arg(1)
	if protocol == "" {
		fmt.Fprintln(os.Stderr, "'connect' option requires the protocol being used directly after 'connect'.")
	}

	host := flag.Arg(2)
	if host == "" {
		fmt.Fprintln(os.Stderr, "'connect' option requires the host being connected to after the protocol.")
	}

	portStr := flag.Arg(3)
	if portStr == "" {
		fmt.Fprintln(os.Stderr, "'connect' option requires the port on the host directly after the host.")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid port '%s': %v\n", portStr, err)
	}
	if port < 0 || port > 65535 {
		fmt.Fprintf(os.Stderr, "invalid port %d; must be within [0, 65535].\n", port)
	}

	if err := client.Connect(protocol, host, port); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
	}
}

func startServer() {
	args := flag.Args()
	if len(args) <= 1 {
		fmt.Fprintln(os.Stderr, "'server' option requires at least one configuration file after 'server'.")
	}

	for i, configPath := range args {
		if i == 0 { continue }

		file, err := os.Open(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s:\n\nfailed to open config file: %v\n", configPath, err)
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

func help(name string) (string, string, func(string)error) {
	showHelpScreen := func() {
		fmt.Fprintln(os.Stdout, "Usage: mp connect PROTOCOL HOST PORT | mp server [CONFIG_FILES...]")
		fmt.Fprintln(os.Stdout, "-h, -help, --h, --help:\n\tshow this help screen")
		fmt.Fprintln(os.Stdout, "-v, -verbose, --v, --verbose:\n\tprint out server logs (not that in depth)")
	}

	return name, "shows this help screen", func(string) error {
		showHelpScreen()
		os.Exit(0)
		return nil
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: mp connect PROTOCOL HOST PORT | mp server [CONFIG_FILES...]")
}
