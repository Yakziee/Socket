package main

import (
	"io"
	"net"
	"os/exec"

	"github.com/creack/pty"
)

type Peer struct {
	conn net.Conn
	app  *App
}

func newPeer(app *App, conn net.Conn) *Peer { return &Peer{conn: conn, app: app} }

func (p *Peer) handleConn() error {
	defer p.app.removePeer(p)

	cmd := exec.Command("bash")

	pty, err := pty.Start(cmd)
	if err != nil {
		p.conn.Close()
		return err
	}

	defer func() {
		p.conn.Close()
		pty.Close()
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		cmd.Wait()
	}()

	errCh := make(chan error, 2)

	go func() {
		_, err := io.Copy(p.conn, pty)
		errCh <- err
	}()

	go func() {
		_, err := io.Copy(pty, p.conn)
		errCh <- err
	}()

	return <-errCh
}
