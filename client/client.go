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

type InputRoute struct {
	Protocol string `json:"protocol"`
	Host string `json:"host"`
	Port string `json:"port"`
}

func Connect(protocol, host string, port, localPort int, outputPort bool) error {
	var conn net.Conn
	var err error

	if localPort >= 0 && localPort <= 65535 {
		localAddr := &net.TCPAddr{
			Port: localPort,
		}
		dialer := net.Dialer{
			LocalAddr: localAddr,
		}

		conn, err = dialer.Dial("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
		if err != nil {
			return fmt.Errorf("failed to connect to %s:%d: %v", localAddr.IP.String(), localAddr.Port, err)
		}
	} else {
		conn, err = net.Dial("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
		if err != nil {
			return fmt.Errorf("failed to connect to %s:%d: %v", host, port, err)
		}
	}
	defer conn.Close()

	if outputPort {
		if err := binary.Write(os.Stdout, binary.BigEndian, uint32(conn.LocalAddr().(*net.TCPAddr).Port)); err != nil {
			return fmt.Errorf("failed to write out client port: %v", err)
		}
	}

	header := InputRoute{
		Protocol: protocol,
		Host:     host,
		Port: strconv.Itoa(port),
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

	_, _ = io.Copy(os.Stdout, conn) // expected to error
	return nil
}
