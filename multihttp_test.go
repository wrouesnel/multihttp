package multihttp

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestParseAddress(t *testing.T) {
	proto, path, err := ParseAddress("unix:///var/test/path.socket")
	assert.Nil(t, err, "Got error from URL parsin")
	assert.Equal(t, "unix", proto)
	
	proto, path, err = ParseAddress("tcp://0.0.0.0:8080")
	assert.Nil(t, err, "Got error from URL parsin")
	assert.Equal(t, "0.0.0.0:8080", path)
}

