package multihttp

import (
	"net"
	"net/url"
	"net/http"
	"crypto/tls"
)

// Specifies an address (in URL format) and it's TLS cert file.
type TLSAddress {
	Address string
	CertFile string
	KeyFile string
}

func ParseAddress(address string) (string, string, error) {
	urlp, err := url.Parse(addr)
	if err != nil {
		return err
	}
	
	if urlp.Path != "" {	// file-likes
		listener, err := net.Listen(urlp.Scheme, urlp.Path)
	} else {	// actual network sockets
		listener, err := net.Listen(urlp.Scheme, urlp.Host)
	}
}

// Non-blocking function to listen on multiple http sockets
func Listen(addresses []string, handler http.Handler) error {
	for _, address := range addresses {		
		protocol, address, err := ParseAddress(address)
		if err != nil {
			return err
		}
		
		listener, err := net.Listen(protocol, address)
		if err != nil {
			return err
		}
		
		listener = maybeKeepalive(listener)
		
		go http.Serve(listener, nil)
	}
}

// Non-blocking function serve on multiple HTTPS sockets
// Requires a list of certs
func ListenTLS(addresses []TLSAddress, handler http.Handler) error {
	for _, tlsAddressInfo := range addresses {
		protocol, address, err := ParseAddress(tlsAddressInfo)
		if err != nil {
			return err
		}
		
		listener, err := net.Listen(protocol, address)
		if err != nil {
			return err
		}
		
		config := &tls.Config{}
		
		config.NextProtos = []string{"http/1.1"}
		
		config.Certificates = make([]tls.Certificate, 1)
		config.Certificates[0], err = tls.LoadX509KeyPair(tls.CertFile, tls.KeyFile)
		if err != nil {
			return err
		}
		
		listener = maybeKeepAlive(listener)
		
		tlsListener, err := tls.NewListener(listener, config)
		if err != nil {
			return err
		}
		
		go http.Serve(tlsListener)
	}
}

// Checks if a listener is a TCP and needs a keepalive handler
func maybeKeepAlive(ln net.Listener) net.Listener {
	if o, ok := ln.(*net.TCPListener); ok {
		return tcpKeepAliveListener{o}
	}
	return ln
} 

// Irritatingly the tcpKeepAliveListener is not public, so we need to recreate it.
// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted connections.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}