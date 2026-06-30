package multihttp

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/url"
	"os"
)

// ParseAddress parses the given address string into an expanded configuration
// struct. It is normally used by the Listen function.
func ParseAddress(address string) (Listener, error) {
	retAddr := Listener{}

	urlp, err := url.Parse(address)
	if err != nil {
		return retAddr, err
	}

	switch NetworkScheme(urlp.Scheme) {
	case NetworkSchemeTcp, NetworkSchemeUnix: // tcp
		retAddr.NetworkType, retAddr.Address = getNetworkTypeAndAddressFromURL(urlp)
	case NetworkSchemeTcps, NetworkSchemeUnixs: // tcp with tls
		urlp.Scheme = urlp.Scheme[:len(urlp.Scheme)-1]
		retAddr.NetworkType, retAddr.Address = getNetworkTypeAndAddressFromURL(urlp)

		tlsConfig := new(tls.Config)
		tlsConfig.NextProtos = []string{"http/1.1"}

		queryParams := urlp.Query()
		if queryParams == nil {
			return retAddr, ErrErrorMissingTLSParameters
		}

		// Get certificate and key path.
		certPath := queryParams.Get(string(ConfigParamTlscert))
		keyPath := queryParams.Get(string(ConfigParamTlskey))

		tlsConfig.Certificates = make([]tls.Certificate, 1)
		tlsConfig.Certificates[0], err = tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return retAddr, errors.Join(ErrErrorLoadingCertificate, err)
		}

		// Optional: client verification path
		if caCertPath := queryParams.Get(string(ConfigParamTlsclientca)); caCertPath != "" {
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert

			// Require acceptable clientCAs to be explicitly specified.
			caCerts, caerr := os.ReadFile(caCertPath)
			if caerr != nil {
				return retAddr, errors.Join(ErrErrorLoadingClientCACertificate, caerr)
			}

			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCerts)

			tlsConfig.ClientCAs = caCertPool
		}
		retAddr.TLSConfig = tlsConfig
	default:
		return retAddr, ErrUnknownListenScheme
	}

	return retAddr, nil
}
