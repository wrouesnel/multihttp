package multihttp

import (
	"net"
	"time"
)

// Checks if a listener is a TCP and needs a keepalive handler.
func maybeKeepAlive(ln net.Listener) net.Listener {
	if o, ok := ln.(*net.TCPListener); ok {
		return &tcpKeepAliveListener{o}
	}
	return ln
}

// Irritatingly the tcpKeepAliveListener is not public, so we need to recreate it.
// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted connections.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}
	err = tc.SetKeepAlive(true)
	if err != nil {
		return nil, err
	}
	err = tc.SetKeepAlivePeriod(3 * time.Minute)
	if err != nil {
		return nil, err
	}
	return tc, nil
}
