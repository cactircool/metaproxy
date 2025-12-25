package server

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/cactircool/metaproxy/client"
)

type OutputRoute struct {
	Host string
	Port int
}

type Routes map[client.InputRoute]OutputRoute

type Config struct {
	ServerPort int
	Routes Routes
}

func ParseConfig(file *os.File) ([]Config, error) {
	skipWhitespace := func(reader *bufio.Reader) error {
		for {
			b, err := reader.ReadByte()
			if err == io.EOF {
				return nil // Reached end of input while skipping
			}
			if err != nil {
				return fmt.Errorf("failed to skip whitespace: %v", err)
			}

			if !unicode.IsSpace(rune(b)) {
				// Found a non-whitespace character, put it back and exit
				if err := reader.UnreadByte(); err != nil {
					return fmt.Errorf("failed to unread byte: %v", err)
				}
				return nil
			}
		}
	}

	readScope := func(opener, closer byte, reader *bufio.Reader) (string, error) {
		var s strings.Builder
		if err := skipWhitespace(reader); err != nil { return "", err }

		if b, err := reader.ReadByte(); b != opener {
			if err != nil {
				return "", err
			}
			return "", fmt.Errorf("expected '%c', got '%c'", opener, b)
		}

		count := 1
		for b, err := reader.ReadByte(); count > 0; b, err = reader.ReadByte() {
			if err == io.EOF {
				return "", fmt.Errorf("EOF encountered before scope terminated.")
			}
			if err != nil {
				return "", fmt.Errorf("failed to read scope: %v", err)
			}

			switch b {
			case opener:
				count++
			case closer:
				count--
			}
			s.WriteByte(b)
		}
		return s.String()[:s.Len() - 1], nil
	}

	expect := func(expected string, reader *bufio.Reader) error {
		if err := skipWhitespace(reader); err != nil { return err }
		var got strings.Builder
		for i := range len(expected) {
			b, err := reader.ReadByte()
			if err != nil {
				return fmt.Errorf("expected '%s', errored with %v", expected, err)
			}
			got.WriteByte(b)

			if b != expected[i] {
				return fmt.Errorf("expected '%s', got '%s'", expected, got.String())
			}
		}
		return nil
	}

	readWord := func(validByte func(byte)bool, reader *bufio.Reader) (string, error) {
		if err := skipWhitespace(reader); err != nil { return "", err }

		var s strings.Builder
		for b, err := reader.ReadByte(); validByte(b); b, err = reader.ReadByte() {
			if err == io.EOF {
				return s.String(), nil
			}
			if err != nil {
				return "", fmt.Errorf("failed to read word: %v", err)
			}

			s.WriteByte(b)
		}
		return s.String(), nil
	}

	readInt := func(reader *bufio.Reader) (int, error) {
		word, err := readWord(func(b byte) bool { return b >= '0' && b <= '9' }, reader)
		if err != nil {
			return -1, fmt.Errorf("expected positive integer: %v", err)
		}
		port, err := strconv.Atoi(word)
		if err != nil {
			return -1, fmt.Errorf("failed to read port: %v", err)
		}
		return port, nil
	}

	reader := bufio.NewReader(file)
	configs := []Config{}

	for {
		if err := skipWhitespace(reader); err != nil {
			return []Config{}, err
		}
		if _, err := reader.Peek(1); err == io.EOF {
			break
		}

		port, err := readInt(reader)
		if err != nil {
			return []Config{}, err
		}

		cfg := Config {
			ServerPort: port,
			Routes: Routes{},
		}

		scope, err := readScope('{', '}', reader)
		if err != nil {
			return []Config{}, err
		}
		scopeReader := bufio.NewReader(strings.NewReader(scope))

		for {
			if err := skipWhitespace(scopeReader); err != nil {
				return []Config{}, err
			}
			if _, err := scopeReader.Peek(1); err == io.EOF {
				break
			}

			input, err := readScope('[', ']', scopeReader)
			if err != nil {
				return []Config{}, err
			}
			inputArgs := strings.Split(input, ";")
			if len(inputArgs) != 2 {
				return []Config{}, fmt.Errorf("expected 3 args in the order and format: [<protocol>;<host>]")
			}

			if err := expect("->", scopeReader); err != nil {
				return []Config{}, err
			}

			output, err := readScope('[', ']', scopeReader)
			if err != nil {
				return []Config{}, err
			}
			outputArgs := strings.Split(output, ";")
			if len(outputArgs) != 2 {
				return []Config{}, fmt.Errorf("expected 2 args in the order and format: [<host>;<port>]")
			}

			outputPort, err := strconv.Atoi(outputArgs[1])
			if err != nil {
				return []Config{}, fmt.Errorf("output port invalid: %v", err)
			}
			cfg.Routes[client.InputRoute{
				Protocol: inputArgs[0],
				Host: inputArgs[1],
			}] = OutputRoute{
				Host: outputArgs[0],
				Port: outputPort,
			}
		}

		configs = append(configs, cfg)
	}

	return configs, nil
}

func Handle(c net.Conn, serverPort int, routes Routes) error {
	defer c.Close()

	fmt.Printf("%s has entered the chat.\n", c.LocalAddr().String())

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

	dest, ok := routes[header]
	if !ok {
		fmt.Printf("Unmapped header detected, forcing %s to leave the chat.\n", c.LocalAddr().String())
		return nil
	}

	target, err := net.Dial("tcp", net.JoinHostPort(dest.Host, strconv.Itoa(dest.Port)))
	if err != nil {
		return fmt.Errorf("failed to forward connection: %v", err)
	}
	defer target.Close()

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
	fmt.Printf("%s has left the chat.\n", c.LocalAddr().String())
	return nil
}

func Start(config Config) error {
	// TODO: make server ipv4 and ipv6
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

	listener, err := net.Listen("tcp", net.JoinHostPort("localhost", strconv.Itoa(config.ServerPort)))
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

		go Handle(conn, config.ServerPort, config.Routes)
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
