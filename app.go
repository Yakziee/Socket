package main

import (
	"net"
	"sync"
)

type App struct {
	mu    sync.Mutex
	peers map[*Peer]struct{}
}

func newApp() *App { return &App{peers: make(map[*Peer]struct{})} }

func (a *App) accept(ln net.Listener) error {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}

		peer := a.addNewPeer(conn)

		go peer.handleConn()
	}
}

func (a *App) addNewPeer(conn net.Conn) *Peer {
	peer := newPeer(a, conn)

	a.mu.Lock()
	a.peers[peer] = struct{}{}
	a.mu.Unlock()

	return peer
}

func (a *App) removePeer(p *Peer) {
	a.mu.Lock()
	delete(a.peers, p)
	a.mu.Unlock()
}
