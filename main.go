package main

import (
	"flag"
	"log"
	"os"

	"github.com/cactircool/metaproxy/client"
	"github.com/cactircool/metaproxy/server"
)

func main() {
	protocol := flag.String("protocol", "", "connection protocol.")
	host := flag.String("host", "", "connection host.")
	port := flag.Int("port", -1, "connection port.")
	config := flag.String("config", "", "server configuration file path.")
	flag.Parse()

	option := flag.Arg(0)
	switch option {
	case "connect":
		if *protocol == "" {
			log.Fatal("'connect' option requires --protocol flag populated with the protocol being used.")
		}

		if *host == "" {
			log.Fatal("'connect' option requires --host flag populated with the host being connected to.")
		}

		if *port == -1 {
			log.Fatal("'connect' option requires --port flag populated with the port on the host being connected to.")
		}

		if *port < 0 || *port > 65535 {
			log.Fatalf("invalid port %d.", *port)
		}

		if err := client.Connect(*protocol, *host, *port); err != nil {
			log.Fatalf("\033[31mfatal\033[0m: %v", err)
		}

	case "server":
		if *config == "" {
			log.Fatal("'server' option requires --config flag populated with a valid file path to a metaproxy configuration file.")
		}

		file, err := os.Open(*config)
		if err != nil {
			log.Fatalf("failed to open config file: %v", err)
		}
		defer file.Close()

		if err := server.ConfigStart(file); err != nil {
			log.Fatalf("failed to parse config and start server: %v", err)
		}

	default:
		log.Fatal("Usage: mp [connect | server] [OPTIONS]")
	}
}
