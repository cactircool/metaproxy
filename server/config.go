package server

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/cactircool/metaproxy/client"
)

type State int

const (
	ROOT State = iota
	INPUT
	OUTPUT
)

type OutputRoute struct {
	Fail bool // Overrides any data in the rest
	Recurse bool
	Host string
	Port int
}

type RoutePair struct {
	Input client.InputRoute
	Output OutputRoute
}

type Routes []RoutePair

type Config struct {
	ServerPort int
	Routes Routes
}

func ParseConfig(file *os.File) ([]Config, error) {
	skipEOF := false
	skipWhitespace := func(reader *bufio.Reader) error {
		for {
			b, err := reader.ReadByte()
			if err == io.EOF {
				skipEOF = true
				return nil // Reached end of input while skipping
			}
			if err != nil {
				return fmt.Errorf("failed to skip whitespace: %v", err)
			}

			if b == '#' {
				_, err := reader.ReadBytes('\n')
				if err == io.EOF {
					skipEOF = true
					return nil
				}
				if err != nil {
					return fmt.Errorf("failed to skip comment: %v", err)
				}
			} else if !unicode.IsSpace(rune(b)) {
				// Found a non-whitespace character, put it back and exit
				if err := reader.UnreadByte(); err != nil {
					return fmt.Errorf("failed to unread byte: %v", err)
				}
				return nil
			}
		}
	}

	condenseMatching := func(valid func(byte)bool, reader *bufio.Reader) (string, error) {
		var s strings.Builder
		for {
			b, err := reader.ReadByte()
			if err == io.EOF {
				return s.String(), nil // Reached end of input while skipping
			}
			if err != nil {
				return "", fmt.Errorf("failed to condense: %v", err)
			}

			if !valid(b) {
				// Found a non-whitespace character, put it back and exit
				if err := reader.UnreadByte(); err != nil {
					return "", fmt.Errorf("failed to unread byte: %v", err)
				}
				return s.String(), nil
			}

			s.WriteByte(b)
		}
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

	reader := bufio.NewReader(file)
	state := ROOT
	configs := []Config{}
	cfg := Config {
		ServerPort: -1,
		Routes: Routes{},
	}
	route := RoutePair {}

	for {
		switch state {
		case ROOT:
			if err := skipWhitespace(reader); err != nil {
				return []Config{}, err
			}

			if skipEOF {
				return configs, nil
			}

			portStr, err := condenseMatching(func(b byte) bool { return b >= '0' && b <= '9' }, reader)
			if err != nil {
				return []Config{}, err
			}

			port, err := strconv.Atoi(portStr)
			if err != nil {
				return []Config{}, fmt.Errorf("invalid port '%s': %v", portStr, err)
			}

			cfg.ServerPort = port

			if err := expect("{", reader); err != nil {
				return []Config{}, err
			}

			state = INPUT
			continue

		case INPUT:
			if err := skipWhitespace(reader); err != nil {
				return []Config{}, err
			}

			b, err := reader.ReadByte()
			if err != nil {
				return []Config{}, err
			}

			if b == '}' {
				state = ROOT
				configs = append(configs, cfg)
				cfg = Config {
					ServerPort: -1,
					Routes: Routes{},
				}
				continue
			} else {
				if err := reader.UnreadByte(); err != nil {
					return []Config{}, err
				}
			}

			if err := expect("[", reader); err != nil {
				return []Config{}, err
			}

			count := 1
			inputStr, err := condenseMatching(func(b byte) bool {
				switch b {
				case '[':
					count++
				case ']':
					count--
				}
				return b != ']' || count != 0
			}, reader)
			if err != nil {
				return []Config{}, err
			}
			if _, err := reader.Discard(1); err != nil {
				return []Config{}, fmt.Errorf("failed to discard ']': %v", err)
			}

			inputArgs := strings.Split(inputStr, ";")
			if len(inputArgs) != 3 {
				return []Config{}, fmt.Errorf("there must be exactly TWO ';' seperating the protocol, the host, and the port on the host it came from (or empty strings).")
			}

			route.Input.Protocol = strings.TrimSpace(inputArgs[0])
			route.Input.Host = strings.TrimSpace(inputArgs[1])
			route.Input.Port = strings.TrimSpace(inputArgs[2])

			if err := expect("->", reader); err != nil {
				return []Config{}, err
			}

			state = OUTPUT
			continue

		case OUTPUT:
			if err := skipWhitespace(reader); err != nil {
				return []Config{}, err
			}

			if b, err := reader.Peek(4); err == nil {
				if string(b) == "fail" {
					if _, err := reader.Discard(4); err != nil {
						return []Config{}, fmt.Errorf("'fail' discard failed: %v", err)
					}
					route.Output.Fail = true
					cfg.Routes = append(cfg.Routes, route)
					route = RoutePair {}
					state = INPUT
					continue
				}
			}

			if b, err := reader.Peek(3); err == nil {
				if string(b) == "rec" {
					if _, err := reader.Discard(3); err != nil {
						return []Config{}, fmt.Errorf("'rec' discard failed: %v", err)
					}
					route.Output.Recurse = true
				}
			}

			if err := expect("[", reader); err != nil {
				return []Config{}, err
			}

			count := 1
			outputStr, err := condenseMatching(func(b byte) bool {
				switch b {
				case '[':
					count++
				case ']':
					count--
				}
				return b != ']' || count != 0
			}, reader)
			if err != nil {
				return []Config{}, err
			}

			if _, err := reader.Discard(1); err != nil {
				return []Config{}, fmt.Errorf("failed to discard ']': %v", err)
			}

			outputArgs := strings.Split(outputStr, ";")
			if len(outputArgs) != 2 {
				return []Config{}, fmt.Errorf("there must be exactly ONE ';' seperating the host and the port (or empty strings).")
			}

			route.Output.Host = strings.TrimSpace(outputArgs[0])
			port, err := strconv.Atoi(strings.TrimSpace(outputArgs[1]))
			if err != nil {
				return []Config{}, fmt.Errorf("invalid port '%s': %v", strings.TrimSpace(outputArgs[1]), err)
			}
			route.Output.Port = port

			cfg.Routes = append(cfg.Routes, route)
			route = RoutePair {}
			state = INPUT
			continue
		}
	}
}
