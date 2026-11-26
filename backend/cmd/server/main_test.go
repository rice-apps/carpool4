package main

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestRequestTimeoutCancelsWork checks that the middleware deadline reaches
// work waiting on the request context.
func TestRequestTimeoutCancelsWork(t *testing.T) {
	t.Parallel()
	var got error
	h := requestTimeoutHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		got = r.Context().Err()
	}), time.Millisecond)

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if !errors.Is(got, context.DeadlineExceeded) {
		t.Errorf("request context error = %v, want %v", got, context.DeadlineExceeded)
	}
}

// TestServeDrainsActiveRequest checks that a deploy does not cut off a response
// already in flight: after cancellation, shutdown begins but serve returns only
// once the held request has been written.
func TestServeDrainsActiveRequest(t *testing.T) {
	t.Parallel()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// One budget for the whole test, so a hang fails the remaining waits at once
	// rather than serially. Sized for a loaded CI runner under -race.
	deadline, cancelDeadline := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelDeadline()

	// Hold one request open while the test asks the server to stop.
	started, release := make(chan struct{}), make(chan struct{})
	shutdownStarted := make(chan struct{})
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		_, _ = w.Write([]byte("finished"))
	})}
	srv.RegisterOnShutdown(func() { close(shutdownStarted) })
	t.Cleanup(func() { _ = srv.Close() })

	done := make(chan error, 1)
	go func() { done <- serve(ctx, srv, listener) }()
	response := make(chan error, 1)
	go func() {
		client := &http.Client{Timeout: 5 * time.Second}
		res, err := client.Get("http://" + listener.Addr().String())
		if err == nil {
			defer res.Body.Close()
			var body []byte
			body, err = io.ReadAll(res.Body)
			if err == nil && string(body) != "finished" {
				err = errors.New("incomplete response")
			}
		}
		response <- err
	}()

	select {
	case <-started:
	case <-deadline.Done():
		t.Fatal("request did not start")
	}

	cancel()
	select {
	case <-shutdownStarted:
	case <-deadline.Done():
		close(release)
		t.Fatal("shutdown did not start")
	}
	select {
	case err := <-done:
		close(release)
		t.Fatalf("serve() returned %v before the held request finished", err)
	default:
	}

	close(release)
	if err := <-response; err != nil {
		t.Errorf("held request failed: %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("serve() error = %v", err)
		}
	case <-deadline.Done():
		t.Error("serve() did not return after the held request finished")
	}
}
