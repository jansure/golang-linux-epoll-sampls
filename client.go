package main

import (
	"fmt"
	"net"
	"time"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Printf("Failed to connect to server: %v\n", err)
		return
	}
	defer conn.Close()

	for i := 0; i < 10; i++ {
		message := fmt.Sprintf("Hello %d\n", i)
		_, err := conn.Write([]byte(message))
		if err != nil {
			fmt.Printf("Failed to send message: %v\n", err)
			return
		}
		time.Sleep(1 * time.Second)
	}
}
