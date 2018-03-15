package multihttp

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type MultiHTTPSuite struct{}

var _ = Suite(&MultiHTTPSuite{})

const (
	// Test CA parsing...
	testCert   = "t/test.testcert.crt"
	testKey    = "t/test.testcert.pem"
	testCAPath = "t/test.ca.crt"
)

func (s *MultiHTTPSuite) TestParseAddress(c *C) {
	addrConfig, err := ParseAddress("unix:///tmp/test.socket")

	c.Assert(err, IsNil)
	c.Assert(addrConfig.NetworkType, Equals, "unix")
	c.Assert(addrConfig.Address, Equals, "/tmp/test.socket")

	addrConfig, err = ParseAddress("tcp://0.0.0.0:8080")

	c.Assert(err, IsNil)
	c.Assert(addrConfig.NetworkType, Equals, "tcp")
	c.Assert(addrConfig.Address, Equals, "0.0.0.0:8080")

	testAddr := fmt.Sprintf("tcps://0.0.0.0:443/?tlscert=%s&tlskey=%s&tlsclientca=%s", testCert, testKey, testCAPath)

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

	listeners, _, err := Listen([]string{fmt.Sprintf("unix://%s", testSocketPath)}, http.NewServeMux())
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

	listeners, _, err := Listen([]string{fmt.Sprintf("tcp://%s", testSocketPath)}, http.NewServeMux())
	c.Assert(err, IsNil)

	for _, listener := range listeners {
		c.Assert(listener.Addr().String(), Equals, testSocketPath)
	}

	CloseAndCleanUpListeners(listeners)
}

func (s *MultiHTTPSuite) TestListenTCPS(c *C) {
	testSocketPath := "127.0.0.1:8443"
	testAddr := fmt.Sprintf("tcps://%s/?tlscert=%s&tlskey=%s&tlsclientca=%s", testSocketPath, testCert, testKey, testCAPath)

	listeners, _, err := Listen([]string{testAddr}, http.NewServeMux())
	c.Assert(err, IsNil)

	for _, listener := range listeners {
		c.Assert(listener.Addr().String(), Equals, testSocketPath)
	}

	CloseAndCleanUpListeners(listeners)
}
