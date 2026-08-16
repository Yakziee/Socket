package main

import (
	"log"
	"os"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	var arg string
	if len(os.Args) > 1 {
		arg = os.Args[1]
	}

	addr, err := parseAddress(arg)
	if err != nil {
		return err
	}

	server, err := newServer()
	if err != nil {
		return err
	}

	app := newApp()
	server.listenAddr = addr

	return server.listen(app)
}
