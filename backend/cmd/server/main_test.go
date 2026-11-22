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
	var got error
	h := requestTimeoutHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		got = r.Context().Err()
	}), time.Millisecond)
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if !errors.Is(got, context.DeadlineExceeded) {
		t.Fatalf("request error = %v, want deadline exceeded", got)
	}
}

// TestServeDrainsActiveRequest checks that signal cancellation waits for an
// in-flight response before serve returns.
func TestServeDrainsActiveRequest(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
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
	case <-time.After(5 * time.Second):
		t.Fatal("request did not start")
	}
	// Shutdown should begin, but serving must wait for the held response.
	cancel()
	select {
	case <-shutdownStarted:
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("shutdown did not start")
	}
	select {
	case err := <-done:
		close(release)
		t.Fatalf("server stopped before request finished: %v", err)
	default:
	}
	// Release the request and confirm the response finishes before shutdown.
	close(release)
	if err := <-response; err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not stop")
	}
}
