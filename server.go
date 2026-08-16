package main

import (
	"crypto/tls"
	"net"
)

type Server struct {
	conf       *tls.Config
	listenAddr string
	ln         net.Listener
}

func newServer() (*Server, error) {
	conf, err := createCertificates()
	if err != nil {
		return nil, err
	}

	return &Server{conf: conf}, nil
}

func (s *Server) listen(a *App) error {
	ln, err := tls.Listen("tcp", s.listenAddr, s.conf)
	if err != nil {
		return err
	}
	defer ln.Close()

	s.ln = ln

	return a.accept(ln)
}
