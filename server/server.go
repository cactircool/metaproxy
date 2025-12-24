package server

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/cactircool/metaproxy/client"
)

type RouteResult struct {
	Host string
	Port int
}

type Routes map[client.Header]RouteResult

type Config struct {
	ServerPort int
	Routes Routes
}

func ParseConfig(file *os.File) ([]Config, error) {
	// TODO: complete (parse a file and populate the routes)
	return []Config{}, nil
}

func Handle(c net.Conn, routes Routes) error {
	defer c.Close()

	var headerLen uint32
	if err := binary.Read(c, binary.BigEndian, &headerLen); err != nil {
		return fmt.Errorf("failed to read header length: %v", err)
	}

	headerBytes := make([]byte, headerLen)
	if _, err := io.ReadFull(c, headerBytes); err != nil {
		return fmt.Errorf("failed to read header: %v", err)
	}

	var header client.Header
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return fmt.Errorf("failed to parse header: %v", err)
	}

	dest, ok := routes[header]
	if !ok {
		dest = RouteResult{
			Host: header.Host,
			Port: header.Port,
		}
	}

	target, err := net.Dial("tcp", net.JoinHostPort(dest.Host, strconv.Itoa(dest.Port)))
	if err != nil {
		return fmt.Errorf("failed to forward connection: %v", err)
	}
	defer target.Close()

	go func() {
		if _, err := io.Copy(target, c); err != nil {
			log.Fatalf("io.Copy(target, c) failed: %v", err)
		}
	}()
	go func() {
		if _, err := io.Copy(c, target); err != nil {
			log.Fatalf("io.Copy(c, target) failed: %v", err)
		}
	}()
	return nil
}

func Start(config Config) error {
	isPortAvailable := func(port int) bool {
		// Format the address as "tcp4" to bind to the IPv4 loopback interface
		// "" as the host means binding to all available interfaces
		address := net.JoinHostPort("", strconv.Itoa(port))
		listener, err := net.Listen("tcp4", address)

		if err != nil {
			return false
		}
		defer listener.Close()
		return true
	}

	if config.ServerPort < 0 || config.ServerPort > 65535 {
		return fmt.Errorf("invalid port %d.", config.ServerPort)
	}

	if !isPortAvailable(config.ServerPort) {
		return fmt.Errorf("cannot bind to port %d.", config.ServerPort)
	}

	listener, err := net.Listen("tcp", strconv.Itoa(config.ServerPort))
	if err != nil {
		return fmt.Errorf("failed to listen on %d: %v", config.ServerPort, err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v\n", err)
			continue
		}

		go Handle(conn, config.Routes)
	}
}

func ConfigStart(file *os.File) error {
	configs, err := ParseConfig(file)
	if err != nil {
		return err
	}

	for _, config := range configs {
		go func() {
			if err := Start(config); err != nil {
				log.Fatalf("failed to start server on port %d: %v", config.ServerPort, err)
			}
		}()
	}
	return nil
}
