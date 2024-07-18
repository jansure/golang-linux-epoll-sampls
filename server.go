//go:build linux
// +build linux

package main

import (
	"fmt"
	"golang.org/x/sys/unix"
	"log"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal("Failed to listen on port 8080： %v", err)
	}
	defer listener.Close()

	listenerFs, err := listener.(*net.TCPListener).File()
	if err != nil {
		log.Fatal("Failed to get file descriptor fd %v", err)
	}
	defer listenerFs.Close()

	epfs, err := unix.EpollCreate1(0)
	if err != nil {
		log.Fatal("Failed to create epoll %v", err)
	}
	defer unix.Close(epfs)

	ev := &unix.EpollEvent{
		Events: unix.EPOLLIN,
		Fd:     int32(listenerFs.Fd()),
	}
	err = unix.EpollCtl(epfs, unix.EPOLL_CTL_ADD, int(listenerFs.Fd()), ev)
	if err != nil {
		log.Fatal("Failed to add listener fs to epoll %v", err)
	}

	events := make([]unix.EpollEvent, 10)
	connections := make(map[int]net.Conn)
	for {
		n, err := unix.EpollWait(epfs, events, -1)
		if err != nil {
			log.Fatal("Failed to wait epoll %v", err)
		}
		for i := 0; i < n; i++ {
			if events[i].Fd == int32(listenerFs.Fd()) {
				//accept new connection
				conn, err := listener.Accept()
				if err != nil {
					log.Fatal("Failed to accept connection %v", err)
					continue
				}

				connFd, err := conn.(*net.TCPConn).File()
				if err != nil {
					log.Println("Failed to get file descriptor %v", err)
					continue
				}

				connections[int(connFd.Fd())] = conn

				ev := &unix.EpollEvent{
					Events: unix.EPOLLIN,
					Fd:     int32(connFd.Fd()),
				}
				err = unix.EpollCtl(epfs, unix.EPOLL_CTL_ADD, int(connFd.Fd()), ev)
				if err != nil {
					log.Fatal("Failed to add connection fs to epoll %v", err)
					conn.Close()
					delete(connections, int(connFd.Fd()))
				}
				fmt.Printf("Accept connection from %v\n", conn.RemoteAddr())
			} else {
				conn := connections[int(events[i].Fd)]
				buf := make([]byte, 1024)
				n, err := conn.Read(buf)
				if err != nil {
					log.Println("Failed to read connection %v", err)
					unix.EpollCtl(epfs, unix.EPOLL_CTL_DEL, int(events[i].Fd), nil)
					conn.Close()
					delete(connections, int(events[i].Fd))
					continue
				}
				message := string(buf[:n])
				fmt.Printf("received message from %v\n", message)
			}
		}
	}
}
