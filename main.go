package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
)

type Server struct {
	Conf       *tls.Config
	listenAddr string
	ln         net.Listener
}

type Peer struct {
	conn net.Conn
	app  *App
}

type App struct {
	mu    sync.Mutex
	peers []*Peer
}

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

	if err := server.listen(app); err != nil {
		return err
	}

	return nil
}

// initialization functions

func newServer() (*Server, error) {
	conf, err := createCertificates()
	if err != nil {
		return nil, err
	}

	return &Server{Conf: conf}, nil
}

func newPeer(app *App, conn net.Conn) *Peer { return &Peer{conn: conn, app: app} }

func newApp() *App { return &App{} }

// helpers/general functions

func parseAddress(arg string) (string, error) {
	const defaultAddress = ":7777"

	arg = strings.TrimSpace(arg)

	if arg == "" {
		return defaultAddress, nil
	}

	if !strings.Contains(arg, ":") {
		arg = ":" + arg
	}

	_, port, err := net.SplitHostPort(arg)
	if err != nil {
		return "", fmt.Errorf("Invalid address %q %w", arg, err)
	}

	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return "", fmt.Errorf("Invalid port: %q", port)
	}

	return arg, nil
}

func send(conn net.Conn, pty *os.File) error {
	_, err := io.Copy(pty, conn)
	return err
}

func receive(conn net.Conn, pty *os.File) error {
	_, err := io.Copy(conn, pty)
	return err
}

func getString() (string, error) {
	b := make([]byte, 10)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return base64.RawStdEncoding.EncodeToString(b), nil
}

func createCertificates() (*tls.Config, error) {
	caKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	certificate := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "CA"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	bytes, err := x509.CreateCertificate(rand.Reader, &certificate, &certificate, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, err
	}

	caCertificate, err := x509.ParseCertificate(bytes)
	if err != nil {
		return nil, err
	}

	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	serverCertificate := x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "server"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	sBytes, err := x509.CreateCertificate(rand.Reader, &serverCertificate, caCertificate, &serverKey.PublicKey, caKey)
	if err != nil {
		return nil, err
	}

	sCertificatePEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: sBytes})
	sKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(serverKey)})
	serverCert, err := tls.X509KeyPair(sCertificatePEM, sKeyPEM)
	if err != nil {
		return nil, err
	}

	clientKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	clientCertificate := x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject:      pkix.Name{CommonName: "c"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	cBytes, err := x509.CreateCertificate(rand.Reader, &clientCertificate, caCertificate, &clientKey.PublicKey, caKey)
	if err != nil {
		return nil, err
	}

	clientCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cBytes})
	cKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(clientKey)})

	os.WriteFile("client.crt", clientCertPEM, 0644)
	os.WriteFile("client.key", cKeyPEM, 0600)

	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: bytes})
	os.WriteFile("ca.crt", caPEM, 0644)

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caPEM)

	return &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caCertPool,
		MinVersion:   tls.VersionTLS12,
	}, nil
}

// Server methods

func (s *Server) listen(a *App) error {
	ln, err := tls.Listen("tcp", s.listenAddr, s.Conf)
	if err != nil {
		return err
	}
	defer ln.Close()

	s.ln = ln

	return a.accept(ln)
}

// App methods

func (a *App) accept(ln net.Listener) error {
	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}

		peer := a.addNewPeer(conn)

		go peer.handleConn()
	}
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

// Peer methods

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
		errCh <- send(p.conn, pty)
	}()

	go func() {
		errCh <- receive(p.conn, pty)
	}()

	return <-errCh
}
