package multihttp

import (
	"context"
	"crypto/tls"
	"net"
	"net/url"
)

// Listener is the parsed form of a multihttp address.
type Listener struct {
	// NetworkType is the type of socket connection
	NetworkType string
	// Address is either the IP or socket path.
	Address string
	// TLS config is the TLS parameters passed as part of the URL.
	TLSConfig *tls.Config
}

func NewFromURL(u *url.URL) (*Listener, error) {
	config := new(ListenerConfig)
	if err := config.DecodeFromURL(u); err != nil {
		return nil, err
	}

	networkType := NetworkScheme(u.Scheme)
	if !networkType.IsValid() {
		return nil, ErrUnknownListenScheme
	}

	var address string

	switch networkType {
	case NetworkSchemeUnix, NetworkSchemeUnixs:
		address = u.Host
	case NetworkSchemeTcp, NetworkSchemeTcps:
		if u.Opaque != "" {
			address = u.Opaque
		} else {
			address = u.Path
		}
	default:
		return nil, ErrUnknownListenScheme
	}

	var tlsConfig *tls.Config

	switch networkType {
	case NetworkSchemeTcps, NetworkSchemeUnixs:
		tlsConfig = &*
	default:
		tlsConfig = nil
	}

	return &Listener{
		NetworkType: string(networkType),
		Address:     address,
		TLSConfig:   tlsConfig,
	}, nil
}

// Listen starts a listener for the given configuration, and closes it when the provided
// context is cancelled.
func (l *Listener) Listen(ctx context.Context) (net.Listener, error) {
	listener, err := net.Listen(l.NetworkType, l.Address)
	if err != nil {
		return nil, err
	}

	if l.TLSConfig != nil {
		listener = tls.NewListener(listener, l.TLSConfig)
	}

	listener = maybeKeepAlive(listener)
	return listener, err
}
