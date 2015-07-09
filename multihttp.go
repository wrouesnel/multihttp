package multihttp

import (
	"net"
	"net/url"
	"net/http"
	"crypto/tls"
	"time"
)

// Specifies an address (in URL format) and it's TLS cert file.
type TLSAddress struct {
	Address string
	CertFile string
	KeyFile string
}

func ParseAddress(address string) (string, string, error) {
	urlp, err := url.Parse(address)
	if err != nil {
		return "", "", err
	}
	
	if urlp.Path != "" {	// file-likes
		return urlp.Scheme, urlp.Path, nil
	} else {	// actual network sockets
		return urlp.Scheme, urlp.Host, nil
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
		
		listener = maybeKeepAlive(listener)
		
		go http.Serve(listener, nil)
	}
	
	return nil
}

// Non-blocking function serve on multiple HTTPS sockets
// Requires a list of certs
func ListenTLS(addresses []TLSAddress, handler http.Handler) error {
	for _, tlsAddressInfo := range addresses {
		protocol, address, err := ParseAddress(tlsAddressInfo.Address)
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
		config.Certificates[0], err = tls.LoadX509KeyPair(tlsAddressInfo.CertFile, tlsAddressInfo.KeyFile)
		if err != nil {
			return err
		}
		
		listener = maybeKeepAlive(listener)
		
		tlsListener := tls.NewListener(listener, config)
		if err != nil {
			return err
		}
		
		go http.Serve(tlsListener, nil)
	}
	
	return nil
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