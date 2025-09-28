package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type MessageType string

const (
	Connect MessageType = "connect"
	Data    MessageType = "data"
	Ack     MessageType = "ack"
	Close   MessageType = "close"
)

type LRCPClient struct {
	conn         net.PacketConn
	serverAddr   *net.UDPAddr
	sessionToken int
	isConnected  bool
	recvBuffer   []byte
	sentPos      int
	recvPos      int
}

func NewLRCPClient(serverAddr string) (*LRCPClient, error) {
	addr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve server address: %v", err)
	}

	conn, err := net.ListenPacket("udp", ":0")
	if err != nil {
		return nil, fmt.Errorf("failed to create UDP socket: %v", err)
	}

	return &LRCPClient{
		conn:       conn,
		serverAddr: addr,
		recvBuffer: make([]byte, 1024),
	}, nil
}

func (c *LRCPClient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *LRCPClient) sendMessage(msg string) error {
	if len(msg) > 1000 {
		return fmt.Errorf("message too large: %d bytes (max 1000)", len(msg))
	}

	_, err := c.conn.WriteTo([]byte(msg), c.serverAddr)
	return err
}

func (c *LRCPClient) receiveMessage(timeout time.Duration) (string, error) {
	c.conn.SetReadDeadline(time.Now().Add(timeout))
	n, addr, err := c.conn.ReadFrom(c.recvBuffer)
	if err != nil {
		return "", err
	}

	if addr.String() != c.serverAddr.String() {
		return "", fmt.Errorf("received message from unexpected source: %s", addr)
	}

	return string(c.recvBuffer[:n]), nil
}

func (c *LRCPClient) Connect(sessionToken int) error {
	if c.isConnected {
		return fmt.Errorf("already connected to session %d", c.sessionToken)
	}

	msg := fmt.Sprintf("/%s/%d/", Connect, sessionToken)

	fmt.Printf("Sending: %s\n", msg)

	for i := range 3 { // Try 3 times
		err := c.sendMessage(msg)
		if err != nil {
			return fmt.Errorf("failed to send connect message: %v", err)
		}

		response, err := c.receiveMessage(3 * time.Second)
		if err == nil {
			fmt.Printf("Received: %s\n", response)

			// Parse response
			if strings.HasPrefix(response, fmt.Sprintf("/%s/%d/", Ack, sessionToken)) {
				c.sessionToken = sessionToken
				c.isConnected = true
				c.sentPos = 0
				c.recvPos = 0
				return nil
			} else {
				return fmt.Errorf("unexpected response: %s", response)
			}
		}

		if i < 2 {
			fmt.Printf("No response received, retrying... (%d/3)\n", i+2)
			time.Sleep(1 * time.Second)
		}
	}

	return fmt.Errorf("connection failed after 3 attempts")
}

func (c *LRCPClient) SendData(data string) error {
	if !c.isConnected {
		return fmt.Errorf("not connected to any session")
	}

	// Escape special characters
	escapedData := escapeData(data)
	msg := fmt.Sprintf("/%s/%d/%d/%s/", Data, c.sessionToken, c.sentPos, escapedData)

	fmt.Printf("Sending: %s\n", msg)

	err := c.sendMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to send data: %v", err)
	}

	// Wait for acknowledgment
	response, err := c.receiveMessage(3 * time.Second)
	if err != nil {
		return fmt.Errorf("failed to receive acknowledgment: %v", err)
	}

	fmt.Printf("Received: %s\n", response)

	// Parse acknowledgment
	if strings.HasPrefix(response, fmt.Sprintf("/%s/%d/", Ack, c.sessionToken)) {
		parts := strings.Split(response, "/")
		if len(parts) >= 4 {
			ackPos, err := strconv.Atoi(parts[3])
			if err == nil && ackPos >= c.sentPos+len(data) {
				c.sentPos = ackPos
				return nil
			}
		}
	}

	return fmt.Errorf("unexpected acknowledgment: %s", response)
}

func (c *LRCPClient) CloseSession() error {
	if !c.isConnected {
		return fmt.Errorf("not connected to any session")
	}

	msg := fmt.Sprintf("/%s/%d/", Close, c.sessionToken)

	fmt.Printf("Sending: %s\n", msg)

	err := c.sendMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to send close message: %v", err)
	}

	// Wait for response
	response, err := c.receiveMessage(3 * time.Second)
	if err != nil {
		return fmt.Errorf("failed to receive close response: %v", err)
	}

	fmt.Printf("Received: %s\n", response)

	if response == msg {
		c.isConnected = false
		return nil
	}

	return fmt.Errorf("unexpected close response: %s", response)
}

func escapeData(data string) string {
	data = strings.ReplaceAll(data, "\\", "\\\\")
	data = strings.ReplaceAll(data, "/", "\\/")
	return data
}

func printHelp() {
	fmt.Println("LRCP REPL Client Commands:")
	fmt.Println("  connect <session_token> - Connect to server with session token")
	fmt.Println("  data <text>            - Send data to server")
	fmt.Println("  close                  - Close current session")
	fmt.Println("  help                   - Show this help message")
	fmt.Println("  quit                   - Exit the REPL")
	fmt.Println()
	fmt.Println("Example usage:")
	fmt.Println("  connect 12345")
	fmt.Println("  data hello world!")
	fmt.Println("  close")
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run lrcp-repl-client.go <server_address:port>")
		fmt.Println("Example: go run lrcp-repl-client.go localhost:8000")
		os.Exit(1)
	}

	serverAddr := os.Args[1]
	client, err := NewLRCPClient(serverAddr)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	fmt.Printf("LRCP REPL Client connected to %s\n", serverAddr)
	printHelp()
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("lrcp> ")
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		command := parts[0]

		switch command {
		case "help":
			printHelp()

		case "quit", "exit":
			if client.isConnected {
				fmt.Println("Closing current session...")
				if err := client.CloseSession(); err != nil {
					fmt.Printf("Error closing session: %v\n", err)
				}
			}
			fmt.Println("Goodbye!")
			return

		case "connect":
			if len(parts) < 2 {
				fmt.Println("Usage: connect <session_token>")
				continue
			}

			sessionToken, err := strconv.Atoi(parts[1])
			if err != nil {
				fmt.Printf("Invalid session token: %s\n", parts[1])
				continue
			}

			if err := client.Connect(sessionToken); err != nil {
				fmt.Printf("Connection failed: %v\n", err)
			} else {
				fmt.Printf("Successfully connected to session %d\n", sessionToken)
			}

		case "data":
			if !client.isConnected {
				fmt.Println("Error: Not connected to any session. Use 'connect' first.")
				continue
			}

			if len(parts) < 2 {
				fmt.Println("Usage: data <text>")
				continue
			}

			data := strings.Join(parts[1:], " ")
			if err := client.SendData(data); err != nil {
				fmt.Printf("Failed to send data: %v\n", err)
			} else {
				fmt.Println("Data sent successfully")
			}

		case "close":
			if !client.isConnected {
				fmt.Println("Error: Not connected to any session.")
				continue
			}

			if err := client.CloseSession(); err != nil {
				fmt.Printf("Failed to close session: %v\n", err)
			} else {
				fmt.Println("Session closed successfully")
			}

		default:
			fmt.Printf("Unknown command: %s. Type 'help' for available commands.\n", command)
		}

		fmt.Println()
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading input: %v\n", err)
	}
}
