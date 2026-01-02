package server

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"regexp"
	"strconv"

	"github.com/cactircool/metaproxy/client"
	"github.com/cactircool/metaproxy/util"
)

func findDestination(header client.InputRoute, routes Routes) (OutputRoute, bool, error) {
	cmp := func(regex, application string) (bool, error) {
		if regex == "" {
			return true, nil
		}
		re, err := regexp.Compile(regex)
		if err != nil {
			return false, fmt.Errorf("failed to compile regex '%s': %v", application, err)
		}

		re.Longest()

		match := re.FindString(application)
		return match == application, nil
	}

	for _, route := range routes {
		found, err := cmp(route.Input.Protocol, header.Protocol)
		if err != nil {
			return OutputRoute{}, false, err
		}
		if !found {
			continue
		}

		found, err = cmp(route.Input.Host, header.Host)
		if err != nil {
			return OutputRoute{}, false, err
		}
		if !found {
			continue
		}

		found, err = cmp(route.Input.Port, header.Port)
		if err != nil {
			return OutputRoute{}, false, err
		}
		if !found {
			continue
		}

		return route.Output, true, nil
	}
	return OutputRoute{}, false, nil
}

func Handle(c net.Conn, serverPort int, routes Routes) error {
	defer c.Close()

	var headerLen uint32
	if err := binary.Read(c, binary.BigEndian, &headerLen); err != nil {
		return fmt.Errorf("failed to read header length: %v", err)
	}

	headerBytes := make([]byte, headerLen)
	if _, err := io.ReadFull(c, headerBytes); err != nil {
		return fmt.Errorf("failed to read header: %v", err)
	}

	var header client.InputRoute
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return fmt.Errorf("failed to parse header: %v", err)
	}

	dest, found, err := findDestination(header, routes)
	if err != nil {
		return fmt.Errorf("failed to find destination: %v", err)
	}

	if !found {
		return fmt.Errorf("unmapped header detected, forcing sudoku")
	}

	if dest.Fail {
		return fmt.Errorf("explicit fail path matched with '%s;%s', forcing sudoku", header.Protocol, header.Host)
	}

	target, err := net.Dial("tcp", net.JoinHostPort(dest.Host, strconv.Itoa(dest.Port)))
	if err != nil {
		return fmt.Errorf("failed to forward connection: %v", err)
	}
	defer target.Close()

	if dest.Recurse {
		headerBytes, err := json.Marshal(header)
		if err != nil {
			return fmt.Errorf("failed to marshal header: %w", err)
		}

		if err := binary.Write(target, binary.BigEndian, uint32(len(headerBytes))); err != nil {
			return fmt.Errorf("failed to write header length: %w", err)
		}

		if _, err := target.Write(headerBytes); err != nil {
			return fmt.Errorf("failed to write header: %w", err)
		}
	}

	done := make(chan struct{}, 2)
	go func() {
		io.Copy(target, c)
		done <- struct{}{}
	}()
	go func() {
		io.Copy(c, target)
		done <- struct{}{}
	}()

	<-done
	return nil
}

func Start(config Config) error {
	isPortAvailable := func(port int) bool {
		// "" as the host means binding to all available interfaces
		address := net.JoinHostPort("", strconv.Itoa(port))
		listener, err := net.Listen("tcp", address)

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

	go func() {
		if err := startListener("tcp4", config); err != nil {
			fmt.Fprintf(os.Stderr, "ipv4 listener error on port %d: %v\n", config.ServerPort, err)
		}
	}()

	go func() {
		if err := startListener("tcp6", config); err != nil {
			fmt.Fprintf(os.Stderr, "ipv6 listener error on port %d: %v\n", config.ServerPort, err)
		}
	}()
	select{}
}

func startListener(network string, config Config) error {
	listener, err := net.Listen(network, net.JoinHostPort("", strconv.Itoa(config.ServerPort)))
	if err != nil {
		return fmt.Errorf("failed to listen on %d: %v", config.ServerPort, err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error accepting connection: %v\n", err)
			continue
		}

		go func() {
			util.Logf(os.Stdout, "%s has entered the chat\n", conn.LocalAddr().String())
			if err := Handle(conn, config.ServerPort, config.Routes); err != nil {
				util.Logf(os.Stderr, "%s -> %v\n", conn.LocalAddr().String(), err)
			}
			util.Logf(os.Stdout, "%s has left the chat\n", conn.LocalAddr().String())
		}()
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
				fmt.Fprintf(os.Stderr, "failed to start server on port %d: %v", config.ServerPort, err)
			}
		}()
	}
	return nil
}
