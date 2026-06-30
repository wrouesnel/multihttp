//go:generate go tool go-enum --marshal --names --values
package multihttp

import (
	"net/url"
	"reflect"

	"github.com/chigopher/pathlib"
	"github.com/gorilla/schema"
)

// NetworkScheme is the scheme which can be provided to a configuration URL
// ENUM(
// unix
// unixs
// tcp
// tcps
// )
type NetworkScheme string

type ListenerConfig struct {
	TLSCertFile           *pathlib.Path `schema:"tlscert"`
	TLSKeyFile            *pathlib.Path `schema:"tlskey"`
	TLSClientCAFile       *pathlib.Path `schema:"tlsclientca"`
	TLSClientCADir        *pathlib.Path `schema:"tlsclientcadir"`
	// TLSClientCertRequest makes an optional client certificate request to connecting
	// clients when no TLS client CA is set. This allows clients which implement TLS
	// auth to receive certificates.
	TLSClientCertRequest bool          `schema:tlsclientcertrequest`
}

func (l *ListenerConfig) DecodeFromURL(u *url.URL) error {
	if l == nil {
		l = new(ListenerConfig)
	}

	decoder := schema.NewDecoder()
	decoder.RegisterConverter(&pathlib.Path{}, func(s string) reflect.Value {
		r := pathlib.NewPath(s)
		return reflect.ValueOf(r)
	})
	decoder.ZeroEmpty(false)

	if err := decoder.Decode(l, u.Query()); err != nil {
		return err
	}
	return nil
}
