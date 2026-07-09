package multihttp //nolint: typecheck

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/wrouesnel/certutils"
	. "gopkg.in/check.v1"
	"resty.dev/v3"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type MultiHTTPSuite struct {
	cARootPath string
	cARootKey  string

	serverCertPath string
	serverKeyPath  string

	clientCARootPath string
	clientCARootKey  string

	clientCertPath string
	clientKeyPath  string
}

var _ = Suite(&MultiHTTPSuite{})

func (s *MultiHTTPSuite) SetUpSuite(c *C) {
	certsDir := c.MkDir()

	s.cARootPath = filepath.Join(certsDir, "ca.crt")
	s.cARootKey = filepath.Join(certsDir, "ca.key")
	s.clientCARootPath = filepath.Join(certsDir, "client-ca.crt")
	s.clientCARootKey = filepath.Join(certsDir, "client-ca.key")
	s.serverCertPath = filepath.Join(certsDir, "server.crt")
	s.serverKeyPath = filepath.Join(certsDir, "server.key")
	s.clientCertPath = filepath.Join(certsDir, "client.crt")
	s.clientKeyPath = filepath.Join(certsDir, "client.key")

	caCert, caKey := generateTestCA(c)
	lo.Must0(
		os.WriteFile(
			s.cARootPath,
			lo.Must(certutils.EncodeCertificates(caCert)),
			os.FileMode(0600),
		),
	)
	lo.Must0(
		os.WriteFile(
			s.cARootKey,
			lo.Must(certutils.EncodeKeys(caKey)),
			os.FileMode(0600),
		),
	)

	cert, key := generateServerCert(c, caCert, caKey)
	lo.Must0(
		os.WriteFile(
			s.serverCertPath,
			lo.Must(certutils.EncodeCertificates(cert)),
			os.FileMode(0600),
		),
	)
	lo.Must0(
		os.WriteFile(
			s.serverKeyPath,
			lo.Must(certutils.EncodeKeys(key)),
			os.FileMode(0600),
		),
	)

	clientCACert, clientCAKey := generateTestCA(c)
	lo.Must0(
		os.WriteFile(
			s.clientCARootPath,
			lo.Must(certutils.EncodeCertificates(clientCACert)),
			os.FileMode(0600),
		),
	)
	lo.Must0(
		os.WriteFile(
			s.clientCARootKey,
			lo.Must(certutils.EncodeKeys(clientCAKey)),
			os.FileMode(0600),
		),
	)

	clientCert, clientKey := generateClientCert(c, clientCACert, clientCAKey)
	lo.Must0(
		os.WriteFile(
			s.clientCertPath,
			lo.Must(certutils.EncodeCertificates(clientCert)),
			os.FileMode(0600),
		),
	)
	lo.Must0(
		os.WriteFile(
			s.clientKeyPath,
			lo.Must(certutils.EncodeKeys(clientKey)),
			os.FileMode(0600),
		),
	)
}

func (s *MultiHTTPSuite) TestParseAddress(c *C) {
	addrConfig, err := ParseAddress("unix:///tmp/test.socket")

	c.Assert(err, IsNil)
	c.Assert(addrConfig.NetworkType, Equals, "unix")
	c.Assert(addrConfig.Address, Equals, "/tmp/test.socket")

	addrConfig, err = ParseAddress("tcp://0.0.0.0:8080")

	c.Assert(err, IsNil)
	c.Assert(addrConfig.NetworkType, Equals, "tcp")
	c.Assert(addrConfig.Address, Equals, "0.0.0.0:8080")

	testAddr := fmt.Sprintf(
		"tcps://0.0.0.0:443/?tlscert=%s&tlskey=%s&tlsclientca=%s",
		s.serverCertPath,
		s.serverKeyPath,
		s.clientCARootPath,
	)

	addrConfig, err = ParseAddress(testAddr)
	c.Assert(err, IsNil)
	c.Assert(addrConfig.NetworkType, Equals, "tcp")
	c.Assert(addrConfig.Address, Equals, "0.0.0.0:443")
	c.Assert(addrConfig.TLSConfig, NotNil)

	addrConfig, err = ParseAddress("fake://0.0.0.0:8080")
	c.Assert(err, Not(IsNil))
}

func (s *MultiHTTPSuite) TestListenUnix(c *C) {
	testSocketPath := "/tmp/test.socket"

	defer os.Remove(testSocketPath) // nolint: errcheck

	listeners, _, err := Listen(
		[]string{fmt.Sprintf("unix://%s", testSocketPath)},
		http.NewServeMux(),
	)
	c.Assert(err, IsNil)

	for _, listener := range listeners {
		addr := listener.Addr()
		//fmt.Printf("%T : %v", addr, addr)
		switch addr.(type) {
		case *net.UnixAddr:
			_, serr := os.Stat(testSocketPath)
			c.Assert(os.IsNotExist(serr), Equals, false)
		}

	}

	CloseAndCleanUpListeners(listeners)
	// Check the listener socket was cleaned up
	_, err = os.Stat(testSocketPath)
	c.Assert(os.IsNotExist(err), Equals, true)
}

func (s *MultiHTTPSuite) TestListenTCP(c *C) {
	testSocketPath := "127.0.0.1:8080"

	listeners, _, err := Listen(
		[]string{fmt.Sprintf("tcp://%s", testSocketPath)},
		http.NewServeMux(),
	)
	c.Assert(err, IsNil)

	for _, listener := range listeners {
		c.Assert(listener.Addr().String(), Equals, testSocketPath)
	}

	CloseAndCleanUpListeners(listeners)
}

func (s *MultiHTTPSuite) TestListenTCPS(c *C) {
	testSocketPath := "127.0.0.1:8443"
	testAddr := fmt.Sprintf(
		"tcps://%s/?tlscert=%s&tlskey=%s&tlsclientca=%s",
		testSocketPath,
		s.serverCertPath,
		s.serverKeyPath,
		s.clientCARootPath,
	)

	listeners, _, err := Listen([]string{testAddr}, http.NewServeMux())
	c.Assert(err, IsNil)

	for _, listener := range listeners {
		c.Assert(listener.Addr().String(), Equals, testSocketPath)
	}

	CloseAndCleanUpListeners(listeners)
}

// TestListenTCPSWithClientCA starts a TLS listener with a client certificate and tests
// that requests can be made against it using a client certificate.
func (s *MultiHTTPSuite) TestListenTCPSWithClientCADefault(c *C) {
	testSocketPath := "127.0.0.1:0"

	testAddr := fmt.Sprintf(
		"tcps://%s/?tlscert=%s&tlskey=%s&tlsclientca=%s",
		testSocketPath,
		s.serverCertPath,
		s.serverKeyPath,
		s.clientCARootPath,
	)

	listeners, _, err := Listen([]string{testAddr}, http.HandlerFunc(httpEcho))
	c.Assert(err, IsNil)

	// Send an unauthenticated request to the server
	client := resty.New()
	client.SetRootCertificates(s.serverCertPath)

	// Test that we demand a client certificate
	defer client.Close()
	for _, listener := range listeners {
		testURL := fmt.Sprintf("https://%s", listener.Addr().String())
		_, err := client.R().
			SetTimeout(time.Second).
			Get(testURL)
		c.Assert(err, Not(IsNil))
		urlErr, _ := errors.AsType[*url.Error](err)
		c.Assert(urlErr.Err.Error(), Equals, "remote error: tls: certificate required")
	}

	// Add the client certificates
	client.SetCertificateFromFile(s.clientCertPath, s.clientKeyPath)
	for _, listener := range listeners {
		testURL := fmt.Sprintf("https://%s", listener.Addr().String())
		res, err := client.R().
			SetTimeout(time.Second).
			Get(testURL)
		c.Assert(err, IsNil)
		c.Assert(err, IsNil, Commentf("could not make request to test server: %v", testURL))
		c.Assert(res.String(), Equals, "OK")
	}

	CloseAndCleanUpListeners(listeners)
}

// TestListenTCPSWithOptionalClientCA starts a TLS listener requesting a client certificate
// but not validating anything.
func (s *MultiHTTPSuite) TestListenTCPSWithOptionalClientCert(c *C) {
	testSocketPath := "127.0.0.1:0"

	testAddr := fmt.Sprintf(
		"tcps://%s/?tlscert=%s&tlskey=%s&tlsclientcert=request",
		testSocketPath,
		s.serverCertPath,
		s.serverKeyPath,
	)

	listeners, _, err := Listen([]string{testAddr}, http.HandlerFunc(httpEcho))
	c.Assert(err, IsNil)

	// Send an unauthenticated request to the server
	client := resty.New()
	client.SetRootCertificates(s.serverCertPath)

	// Test that requests are accepted without a client cert
	defer client.Close()
	for _, listener := range listeners {
		testURL := fmt.Sprintf("https://%s", listener.Addr().String())
		res, err := client.R().
			SetTimeout(time.Second).
			Get(testURL)
		c.Assert(err, IsNil, Commentf("request without client cert failed: %v", testURL))
		c.Assert(res.String(), Equals, "OK")
	}

	// Test that requests are accepted with a client cert
	client.SetCertificateFromFile(s.clientCertPath, s.clientKeyPath)
	for _, listener := range listeners {
		testURL := fmt.Sprintf("https://%s", listener.Addr().String())
		res, err := client.R().
			SetTimeout(time.Second).
			Get(testURL)
		c.Assert(err, IsNil, Commentf("request with client cert failed: %v", testURL))
		c.Assert(res.String(), Equals, "OK")
	}

	CloseAndCleanUpListeners(listeners)
}

// TestListenTCPSWithOptionalClientCA starts a TLS listener requesting a client certificate
// but not validating anything.
func (s *MultiHTTPSuite) TestListenTCPSWithRequiredClientCert(c *C) {
	testSocketPath := "127.0.0.1:0"

	testAddr := fmt.Sprintf(
		"tcps://%s/?tlscert=%s&tlskey=%s&tlsclientcert=require",
		testSocketPath,
		s.serverCertPath,
		s.serverKeyPath,
	)

	listeners, _, err := Listen([]string{testAddr}, http.HandlerFunc(httpEcho))
	c.Assert(err, IsNil)

	// Send an unauthenticated request to the server
	client := resty.New()
	client.SetRootCertificates(s.serverCertPath)

	// Test that requests are accepted without a client cert
	defer client.Close()
	for _, listener := range listeners {
		testURL := fmt.Sprintf("https://%s", listener.Addr().String())
		_, err := client.R().
			SetTimeout(time.Second).
			Get(testURL)
		c.Assert(err, Not(IsNil))
		urlErr, _ := errors.AsType[*url.Error](err)
		c.Assert(urlErr.Err.Error(), Equals, "remote error: tls: certificate required")
	}

	// Test that requests are accepted with a client cert
	client.SetCertificateFromFile(s.clientCertPath, s.clientKeyPath)
	for _, listener := range listeners {
		testURL := fmt.Sprintf("https://%s", listener.Addr().String())
		res, err := client.R().
			SetTimeout(time.Second).
			Get(testURL)
		c.Assert(err, IsNil, Commentf("request with client cert failed: %v", testURL))
		c.Assert(res.String(), Equals, "OK")
	}

	CloseAndCleanUpListeners(listeners)
}

// Simple handler for testing purposes
func httpEcho(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK\n"))
}

// generateTestCA invents a test CA
func generateTestCA(c *C) (*x509.Certificate, any) {
	// Generate a CA certificate
	privKey := lo.Must(certutils.GeneratePrivateKey(certutils.PrivateKeyTypeEcp256))
	csr := lo.Must(certutils.GenerateCSR(pkix.Name{
		Country:            []string{"TEST"},
		Organization:       []string{"TEST_ORG"},
		OrganizationalUnit: []string{"TEST_OU"},
		CommonName:         c.TestName(),
	}, certutils.CSRParameters{
		KeyUsage: x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		IsCA:     true,
	}, privKey))

	cert := lo.Must(certutils.SignCertificate(csr, nil, privKey, certutils.SigningParameters{
		SerialNumber: time.Now().UnixNano(),
		NotBefore:    certutils.CertificateNotBefore(),
		NotAfter:     certutils.CACertificateNotAfter(0),
	}))

	return cert, privKey
}

// generateServerCert generates a client certificate
func generateServerCert(c *C, ca *x509.Certificate, cakey any) (*x509.Certificate, any) {
	privKey := lo.Must(certutils.GeneratePrivateKey(certutils.PrivateKeyTypeEcp256))
	csr := lo.Must(certutils.GenerateCSR(pkix.Name{
		Country:            []string{"TEST"},
		Organization:       []string{"TEST_ORG"},
		OrganizationalUnit: []string{"TEST_OU"},
		CommonName:         c.TestName(),
	}, certutils.CSRParameters{
		KeyUsage:    x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IsCA:        false,
	}, privKey, "127.0.0.1"))

	cert := lo.Must(certutils.SignCertificate(csr, ca, cakey, certutils.SigningParameters{
		SerialNumber: time.Now().UnixNano(),
		NotBefore:    certutils.CertificateNotBefore(),
		NotAfter:     certutils.CertificateNotAfter(0, ca),
	}))

	return cert, privKey
}

// generateClientCert generates a client certificate
func generateClientCert(c *C, ca *x509.Certificate, cakey any) (*x509.Certificate, any) {
	privKey := lo.Must(certutils.GeneratePrivateKey(certutils.PrivateKeyTypeEcp256))
	csr := lo.Must(certutils.GenerateCSR(pkix.Name{
		Country:            []string{"TEST"},
		Organization:       []string{"TEST_ORG"},
		OrganizationalUnit: []string{"TEST_OU"},
		CommonName:         c.TestName(),
	}, certutils.CSRParameters{
		KeyUsage:    x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		IsCA:        false,
	}, privKey, c.TestName()))

	cert := lo.Must(certutils.SignCertificate(csr, ca, cakey, certutils.SigningParameters{
		SerialNumber: time.Now().UnixNano(),
		NotBefore:    certutils.CertificateNotBefore(),
		NotAfter:     certutils.CertificateNotAfter(0, ca),
	}))

	return cert, privKey
}
