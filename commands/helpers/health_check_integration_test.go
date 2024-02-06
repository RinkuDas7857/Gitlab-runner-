//go:build integration

package helpers

import (
	"context"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-runner/helpers"
)

func TestServiceWaiterCommand_NoEnvironmentVariables(t *testing.T) {
	removeHook := helpers.MakeFatalToPanic()
	defer removeHook()

	// Make sure there are no env vars that match the pattern
	for _, e := range os.Environ() {
		if strings.Contains(e, "_TCP_") {
			err := os.Unsetenv(strings.Split(e, "=")[0])
			require.NoError(t, err)
		}
	}

	cmd := HealthCheckCommand{}

	assert.Panics(t, func() {
		cmd.Execute(nil)
	})
}

func TestHealthCheckCommand_Execute(t *testing.T) {
	cases := []struct {
		name            string
		expectedConnect bool
		exposeHigher    bool
		exposeLower     bool
	}{
		{
			name:            "Successful connect",
			expectedConnect: true,
			exposeHigher:    false,
			exposeLower:     false,
		},
		{
			name:            "Unsuccessful connect because service is down",
			expectedConnect: false,
			exposeHigher:    false,
			exposeLower:     false,
		},
		{
			name:            "Successful connect with higher port exposed",
			expectedConnect: true,
			exposeHigher:    true,
			exposeLower:     false,
		},
		{
			name:            "Successful connect with lower port exposed",
			expectedConnect: true,
			exposeHigher:    false,
			exposeLower:     true,
		},
		{
			name:            "Successful connect with both lower and higher port exposed",
			expectedConnect: true,
			exposeHigher:    true,
			exposeLower:     true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			os.Unsetenv("SERVICE_LOWER_TCP_PORT")
			os.Unsetenv("SERVICE_HIGHER_TCP_PORT")

			// Start listening to reverse addr
			listener, err := net.Listen("tcp", "127.0.0.1:")
			require.NoError(t, err)
			defer listener.Close()

			port := listener.Addr().(*net.TCPAddr).Port

			err = os.Setenv("SERVICE_TCP_ADDR", "127.0.0.1")
			require.NoError(t, err)

			err = os.Setenv("SERVICE_TCP_PORT", strconv.Itoa(port))
			require.NoError(t, err)

			if c.exposeHigher {
				err = os.Setenv("SERVICE_HIGHER_TCP_PORT", strconv.Itoa(port+1))
				require.NoError(t, err)
			}

			if c.exposeLower {
				err = os.Setenv("SERVICE_LOWER_TCP_PORT", strconv.Itoa(port-1))
				require.NoError(t, err)
			}

			// If we don't expect to connect we close the listener.
			if !c.expectedConnect {
				listener.Close()
			}

			ctx, cancelFn := context.WithTimeout(context.Background(), 4*time.Second)
			defer cancelFn()
			done := make(chan struct{})
			go func() {
				cmd := HealthCheckCommand{ctx: ctx}
				cmd.Execute(nil)
				done <- struct{}{}
			}()

			select {
			case <-ctx.Done():
				if c.expectedConnect {
					require.Fail(t, "Timeout waiting to start service.")
				}
			case <-done:
				if !c.expectedConnect {
					require.Fail(t, "Expected to not connect to server")
				}
			}
		})
	}
}

func TestHealthCheckCommand_WaitAll(t *testing.T) {
	// We might simulate as many as two services.
	const MAX_PORTS = 2

	cases := []struct {
		name            string
		successCount    int
		expectedTimeout bool
	}{
		{
			name:            "Two services down",
			successCount:    0,
			expectedTimeout: true,
		},
		{
			name:            "One up one down",
			successCount:    1,
			expectedTimeout: true,
		},
		{
			name:            "Two services up",
			successCount:    2,
			expectedTimeout: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {

			ports := make([]string, 0)

			for i := 0; i < c.successCount; i++ {
				listener, err := net.Listen("tcp", "127.0.0.1:")
				require.NoError(t, err)
				defer listener.Close()

				port := listener.Addr().(*net.TCPAddr).Port
				ports = append(ports, strconv.Itoa(port))
			}

			// To simulate services that are down, find an unused port and increment port
			// numbers from there.
			unusedPort := 0
			if c.successCount < MAX_PORTS {
				listener, err := net.Listen("tcp", "127.0.0.1:")
				require.NoError(t, err)

				unusedPort = listener.Addr().(*net.TCPAddr).Port
				listener.Close()
			}

			for i := c.successCount; i < MAX_PORTS; i++ {
				ports = append(ports, strconv.Itoa(unusedPort))
				unusedPort++
			}

			// The cli package provides an extra value at end of the args array
			ports = append(ports, "[unused value]")

			ctx, cancelFn := context.WithTimeout(context.Background(), 4*time.Second)
			defer cancelFn()
			done := make(chan struct{})
			go func() {
				cmd := HealthCheckCommand{
					ctx:   ctx,
					Ports: ports,
				}
				cmd.Execute(nil)
				done <- struct{}{}
			}()

			select {
			case <-ctx.Done():
				if !c.expectedTimeout {
					require.Fail(t, "Unexpected timeout")
				}
			case <-done:
				if c.expectedTimeout {
					require.Fail(t, "Unexpected failure to time out")
				}
			}
		})
	}
}
