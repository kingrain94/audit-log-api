package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Usage: go run test_websocket_client.go <JWT_TOKEN>")
	}

	token := os.Args[1]
	url := "ws://localhost:10000/api/v1/logs/stream"
	header := http.Header{}
	header.Set("Authorization", "Bearer "+token)
	fmt.Printf("Connecting to %s...\n", url)
	conn, _, err := websocket.DefaultDialer.Dial(url, header)
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer conn.Close()
	conn.SetPingHandler(func(appData string) error {
		fmt.Println("Received ping from server, sending pong")
		return conn.WriteMessage(websocket.PongMessage, nil)
	})

	fmt.Println("Connected! Waiting for log messages...")
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("Read error:", err)
				return
			}
			fmt.Printf("%s\n", string(message))
		}
	}()

	select {
	case <-done:
		return
	case <-interrupt:
		fmt.Println("\nDisconnecting...")

		// Send close message
		err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			log.Println("Write close:", err)
			return
		}

		// Wait for the connection to close
		select {
		case <-done:
		case <-time.After(time.Second):
		}
	}
}
