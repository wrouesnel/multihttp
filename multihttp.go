package multihttp //nolint:typecheck

import (
	"crypto/tls"
	"net"
	"net/http"
	"os"
)

// CloseAndCleanUpListeners runs clean up on a list of listeners,
// namely deleting any Unix socket files
// nolint: errcheck,gas
func CloseAndCleanUpListeners(listeners []net.Listener) {
	for _, listener := range listeners {
		listener.Close()
		addr := listener.Addr()
		switch addr.(type) {
		case *net.UnixAddr:
			os.Remove(addr.String())
		}
	}
}

// Listen is a non-blocking function to listen on multiple sockets. Returns
// a list of the created listener interfaces. Even in the case of errors,
// successfully listening interfaces are returned to allow for clean up.
func Listen(addresses []string, handler http.Handler) ([]net.Listener, <-chan *ListenerError, error) {
	return ListenFunc(addresses, func(listener net.Listener) error {
		return http.Serve(listener, handler)
	})
}

// ListenFunc is a non-blocking function to listen on multiple http sockets. Returns
// a list of the created listener interfaces. Even in the case of errors,
// successfully listening interfaces are returned to allow for clean up.
func ListenFunc(addresses []string, listenFunc func(listener net.Listener) error) ([]net.Listener, <-chan *ListenerError, error) {
	var listeners []net.Listener

	// Master error channel - all errors are propagated here. Length is set to
	// listener length so go routines will clean up even if the channel is
	// ignored.
	errCh := make(chan *ListenerError, len(addresses))

	for _, address := range addresses {
		addressConfig, aerr := ParseAddress(address)
		if aerr != nil {
			return listeners, errCh, aerr
		}

		var listener net.Listener
		var lerr error

		listener, lerr = net.Listen(addressConfig.NetworkType, addressConfig.Address)
		// Errored making listener?
		if lerr != nil {
			return listeners, errCh, lerr
		}

		// TLS connection?
		if addressConfig.TLSConfig != nil {
			listener = tls.NewListener(listener, addressConfig.TLSConfig)
		}

		// Append and start serving on listener
		listener = maybeKeepAlive(listener)
		listeners = append(listeners, listener)

		// Allow specifying a nil handler if the user just wants the listeners.
		if listenFunc != nil {
			go func(listener net.Listener) {
				err := listenFunc(listener)
				if err != nil {
					// Return the listener and the error it returned.
					errCh <- &ListenerError{
						Listener: listener,
						Err:      err,
					}
				}
			}(listener)
		}
	}

	return listeners, errCh, nil
}
