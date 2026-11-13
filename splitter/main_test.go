package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSplitter(t *testing.T) {
	got := make(chan string, 10)
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		io.WriteString(w, "primary")
	}))
	secondary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got <- r.Method + " " + r.URL.RequestURI() + " " + string(b) + " " + r.Header.Get("X-Gotify-Key")
	}))
	defer primary.Close()
	defer secondary.Close()
	pu, _ := url.Parse(primary.URL)
	su, _ := url.Parse(secondary.URL)
	front := httptest.NewServer(newHandler(pu, su))
	defer front.Close()

	// GET: primary only
	resp, _ := http.Get(front.URL + "/message")
	if b, _ := io.ReadAll(resp.Body); resp.StatusCode != 418 || string(b) != "primary" {
		t.Fatalf("client must get the primary response, got %d %q", resp.StatusCode, b)
	}
	// POST: primary answers the client, secondary gets an identical copy
	req, _ := http.NewRequest("POST", front.URL+"/message?x=1", strings.NewReader("hi"))
	req.Header.Set("X-Gotify-Key", "tok")
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != 418 {
		t.Fatalf("got %d", resp.StatusCode)
	}
	select {
	case s := <-got:
		if s != "POST /message?x=1 hi tok" {
			t.Fatalf("mirror got %q (a GET must not be mirrored)", s)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("secondary never received the POST")
	}

	// secondary down: client must be unaffected
	secondary.Close()
	resp, err := http.Post(front.URL+"/message", "text/plain", strings.NewReader("x"))
	if err != nil || resp.StatusCode != 418 {
		t.Fatalf("secondary failure leaked to client: %v %v", resp, err)
	}
}
