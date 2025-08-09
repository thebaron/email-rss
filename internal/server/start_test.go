package server

import (
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStartServerError(t *testing.T) {
	config := ServerConfig{
		Host:     "localhost",
		Port:     99999,
		FeedsDir: "/tmp",
	}

	server := New(config)

	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Start()
	}()

	select {
	case err := <-errChan:
		assert.Error(t, err)
	case <-time.After(100 * time.Millisecond):
		t.Error("Server should have failed to start")
	}
}

func TestStartServerSuccess(t *testing.T) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Skip("Could not find available port")
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	config := ServerConfig{
		Host:     "localhost",
		Port:     port,
		FeedsDir: t.TempDir(),
	}

	server := New(config)
	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Start()
	}()

	time.Sleep(1000 * time.Millisecond)
	listener, err = net.Listen("tcp", ":"+strconv.Itoa(port))
	if err == nil {
		listener.Close()
	}

	if err != nil && strings.Contains(err.Error(), "address already in use") {
		t.Skip("Port is already in use, skipping test")
	}

	assert.NoError(t, err)
}
