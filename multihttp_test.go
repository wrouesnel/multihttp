package multihttp

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "net"
    "net/http"
    //"fmt"
    "os"
    //"reflect"
)

func TestParseAddress(t *testing.T) {
	proto, path, err := ParseAddress("unix:///var/test/path.socket")
	assert.Nil(t, err, "Got error from URL parsing")
	assert.Equal(t, "unix", proto)
	
	proto, path, err = ParseAddress("tcp://0.0.0.0:8080")
	assert.Nil(t, err, "Got error from URL parsing")
	assert.Equal(t, "0.0.0.0:8080", path)
}

func TestListen(t *testing.T) {
	defer os.Remove("/tmp/test.socket")
	
	listeners, err := Listen([]string{"unix:///tmp/test.socket"}, http.NewServeMux())
	assert.Nil(t, err, "Got err from listeners")
	
	for _, listener := range listeners {
		listener.Close()
		addr := listener.Addr()
		//fmt.Printf("%T : %v", addr, addr)
		switch addr.(type) {
			case *net.UnixAddr:
				_, err := os.Stat(addr.String())
				assert.False(t, os.IsNotExist(err), "Unix socket was created")
				os.Remove(addr.String())
		}
		
		_, err := os.Stat(addr.String())
		assert.True(t, os.IsNotExist(err), "Unix socket removed successfully") 
	}
	
	
}