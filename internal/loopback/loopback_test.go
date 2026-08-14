package loopback

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestLoginURLMatchesPorteContract(t *testing.T) {
	server, err := Listen()
	if err != nil {
		t.Fatal(err)
	}
	defer server.Listener.Close()

	raw := server.LoginURL("https://journal.facile.studio", "deadbeef")
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Scheme != "https" || parsed.Host != "journal.facile.studio" {
		t.Fatalf("unexpected origin: %s", parsed.Scheme+"://"+parsed.Host)
	}
	if parsed.Path != "/api/auth/oidc" {
		t.Errorf("path = %q, want /api/auth/oidc", parsed.Path)
	}
	query := parsed.Query()
	// These are the exact parameter names porte reads. Get one wrong and the
	// browser flow silently does not go to CLI mode.
	if query.Get("flow") != "cli" {
		t.Errorf("flow = %q, want cli", query.Get("flow"))
	}
	if query.Get("cli_state") != "deadbeef" {
		t.Errorf("cli_state = %q, want deadbeef", query.Get("cli_state"))
	}
	if query.Get("port") != fmt.Sprintf("%d", server.Port) {
		t.Errorf("port = %q, want %d", query.Get("port"), server.Port)
	}
}

func TestWaitForCodeVerifiesState(t *testing.T) {
	server, err := Listen()
	if err != nil {
		t.Fatal(err)
	}
	defer server.Listener.Close()

	result := make(chan string, 1)
	go func() {
		code, err := server.WaitForCode("expected")
		if err != nil {
			result <- "ERR:" + err.Error()
			return
		}
		result <- code
	}()

	// Wait until the listener is serving, then hit it exactly as porte's
	// redirect would: code plus the echoed state.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		response, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/?code=abc&state=expected", server.Port))
		if err == nil {
			body := make([]byte, 16)
			_, _ = response.Body.Read(body)
			response.Body.Close()
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	select {
	case got := <-result:
		if got != "abc" {
			t.Fatalf("code = %q, want abc", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the code")
	}
}
