// Package loopback runs the one-time listening server that the porte SSO CLI
// flow redirects to. It opens a port, hands the browser a URL, and waits for
// the login code to land on localhost.
package loopback

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"time"
)

const timeout = 90 * time.Second

// Server is a listening loopback socket, bound before the login URL is built so
// the port is known and taken.
type Server struct {
	Listener net.Listener
	Port     int
}

// Listen binds an ephemeral port on loopback.
func Listen() (*Server, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("cannot open the login listener — %w", err)
	}
	return &Server{
		Listener: listener,
		Port:     listener.Addr().(*net.TCPAddr).Port,
	}, nil
}

// RandomState returns a fresh nonce echoed through the login flow, so the
// callback that lands on this port is provably the one this login started.
func RandomState() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("cannot draw a login nonce — %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// LoginURL builds the address the browser starts the flow at.
func (s *Server) LoginURL(base, state string) string {
	query := url.Values{}
	query.Set("flow", "cli")
	query.Set("port", fmt.Sprintf("%d", s.Port))
	query.Set("cli_state", state)
	return fmt.Sprintf("%s/api/auth/oidc?%s", base, query.Encode())
}

// OpenBrowser hands the URL to the OS default browser. It reports whether it
// found one to delegate to.
func OpenBrowser(rawURL string) bool {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.Command("open", rawURL)
	case "windows":
		command = exec.Command("cmd", "/C", "start", "", rawURL)
	default:
		command = exec.Command("xdg-open", rawURL)
	}
	return command.Start() == nil
}

// WaitForCode blocks until the browser completes the login and lands here with
// the code, or the timeout elapses. The returned code is single use.
func (s *Server) WaitForCode(state string) (string, error) {
	type result struct {
		code string
		err  error
	}
	done := make(chan result, 1)

	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		// Every login carries its own nonce; a callback that does not echo
		// ours back belongs to a different flow and is refused.
		if echoed := query.Get("state"); echoed != state {
			http.Error(w, "login attempt did not match", http.StatusBadRequest)
			return
		}
		if code := query.Get("code"); code != "" {
			done <- result{code: code}
		}
		// The browser will not wait on us: say something and let the caller
		// pick up the code. The page body is irrelevant, only the code matters.
		w.Header().Set("Content-Type", "text/plain")
		_, _ = fmt.Fprintln(w, "Signed in. You can close this tab.")
	})}

	go func() {
		err := server.Serve(s.Listener)
		if err != http.ErrServerClosed {
			done <- result{err: err}
		}
	}()

	defer server.Close()

	select {
	case res := <-done:
		if res.err != nil {
			return "", fmt.Errorf("the login listener failed — %w", res.err)
		}
		return res.code, nil
	case <-time.After(timeout):
		return "", fmt.Errorf("timed out waiting for the login to complete")
	}
}
