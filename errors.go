package multihttp

import (
	"errors"
	"fmt"
	"net"
)

// nolint: golint
var (
	ErrErrorLoadingCertificate         = errors.New("could not load TLS certificate")
	ErrErrorMissingTLSParameters       = errors.New("TLS modes require tlscert and tlskey params")
	ErrErrorLoadingClientCACertificate = errors.New("error loading client CA certificate")
	ErrUnknownListenScheme             = errors.New("unknown listen scheme")
)

// ListenerError maps a listener to it's error channel.
type ListenerError struct {
	Listener net.Listener
	Err      error
}

func (e ListenerError) Unwrap() error {
	return e.Err
}

func (e ListenerError) Error() string {
	return fmt.Sprintf("listener %s: %s", e.Listener.Addr().String(), e.Err.Error())
}