package client

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
)

type Header struct {
	Protocol string `json:"protocol"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
}

func Connect(protocol, host string, port int) error {
	conn, err := net.Dial("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return fmt.Errorf("failed to connect to %s:%d: %v", host, port, err)
	}
	defer conn.Close()

	header := Header{
		Protocol: protocol,
		Host:     host,
		Port:     port,
	}

	headerBytes, err := json.Marshal(header)
	if err != nil {
		return fmt.Errorf("failed to marshal header: %w", err)
	}

	if err := binary.Write(conn, binary.BigEndian, uint32(len(headerBytes))); err != nil {
		return fmt.Errorf("failed to write header length: %w", err)
	}

	if _, err := conn.Write(headerBytes); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	go func() {
		_, _ = io.Copy(conn, os.Stdin) // closing is an error so just ignore
		if tcp, ok := conn.(*net.TCPConn); ok {
			tcp.CloseWrite()
		}
	}()

	_, err = io.Copy(os.Stdout, conn)
	if err != nil {
		return fmt.Errorf("io.Copy(os.Stdout, conn) failed: %v", err)
	}
	return nil
}
