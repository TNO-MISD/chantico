package main

import (
    "log"
    "math/rand"
    "net"
    "time"
	"fmt"
	"os"
	"strconv"

    "github.com/gosnmp/gosnmp"
)

func buildRandomSNMPResponse() []byte {
    rand.Seed(time.Now().UnixNano())
    randomValue := rand.Intn(1000)

    response := gosnmp.SnmpPacket{
        Version:   gosnmp.Version2c,
        Community: "public",
        PDUType:   gosnmp.GetResponse,
        Variables: []gosnmp.SnmpPDU{
            {
                Name:  ".1.3.6.1.4.1.2021.255", // An example OID
                Type:  gosnmp.Integer,
                Value: randomValue,
            },
        },
    }

    // Marshal it into BER-encoded SNMP message
    out, err := response.MarshalMsg()
    if err != nil {
        log.Printf("Failed to marshal SNMP response: %v", err)
        return []byte{}
    }

    return out
}

func handleConnection(conn net.Conn) {
    defer conn.Close()

    // Read request (but we ignore it in this mock)
    buf := make([]byte, 2048)
    n, err := conn.Read(buf)
    if err != nil {
        log.Printf("Read error: %v", err)
        return
    }

    log.Printf("Received request (%d bytes)", n)

    // Generate a response with a random value
    response := buildRandomSNMPResponse()

    // Send it back
    _, err = conn.Write(response)
    if err != nil {
        log.Printf("Write error: %v", err)
        return
    }

    log.Printf("Sent mock SNMP response (%d bytes)", len(response))
}

// startServer listens on a specific port.
func startServer(port int) {
    addr := fmt.Sprintf(":%d", port)
    ln, err := net.Listen("tcp", addr)
    if err != nil {
        log.Fatalf("Failed to listen on port %d: %v", port, err)
    }

    log.Printf("SNMockP - Listening on TCP port %d...", port)

    for {
        conn, err := ln.Accept()
        if err != nil {
            log.Printf("Failed to accept connection on port %d: %v", port, err)
            continue
        }

        go handleConnection(conn)
    }
}

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: go run mock_snmp_tcp.go <port1> <port2> ...")
        os.Exit(1)
    }

    for _, arg := range os.Args[1:] {
        port, err := strconv.Atoi(arg)
        if err != nil || port <= 0 || port > 65535 {
            log.Fatalf("Invalid port: %s", arg)
        }

        go startServer(port)
    }

    // Keep the main goroutine alive
    select {}
}
