// Splitter: reverse proxy in front of two Gotify backends (v2 + v3).
// Every request goes to PRIMARY_URL and its response is returned to the client.
// State-changing requests (anything but GET/HEAD/OPTIONS) are also replayed, in order,
// to SECONDARY_URL; its answer is discarded and its failures never reach the client.
// To switch the "source of truth" to v3, swap the two URLs.
package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

const maxBody = 32 << 20 // bodies are buffered to be replayed (app image uploads are the biggest)

func newHandler(primary, secondary *url.URL) http.Handler {
	proxy := httputil.NewSingleHostReverseProxy(primary)
	client := &http.Client{Timeout: 30e9}

	// One worker keeps the writes on the secondary in the same order as on the primary.
	// ponytail: in-memory queue, mirrored writes are lost on restart/overflow; fine for a temporary migration setup.
	queue := make(chan *http.Request, 1024)
	go func() {
		for req := range queue {
			resp, err := client.Do(req)
			if err != nil {
				log.Printf("mirror %s %s: %v", req.Method, req.URL.Path, err)
				continue
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode >= 400 {
				log.Printf("mirror %s %s: secondary answered %d", req.Method, req.URL.Path, resp.StatusCode)
			}
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBody))
			if err != nil {
				http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))

			target := secondary.ResolveReference(&url.URL{Path: r.URL.Path, RawPath: r.URL.RawPath, RawQuery: r.URL.RawQuery})
			// Detached from r.Context(): the mirror must survive the client hanging up.
			if mirror, err := http.NewRequest(r.Method, target.String(), bytes.NewReader(body)); err == nil {
				mirror.Header = r.Header.Clone()
				select {
				case queue <- mirror:
				default:
					log.Printf("mirror queue full, dropped %s %s", r.Method, r.URL.Path)
				}
			}
		}
		proxy.ServeHTTP(w, r) // also handles the /stream websocket (GET), primary only
	})
}

func mustURL(env string) *url.URL {
	u, err := url.Parse(os.Getenv(env))
	if err != nil || u.Host == "" {
		log.Fatalf("%s must be a full URL, e.g. http://gotify:80", env)
	}
	return u
}

func main() {
	listen := os.Getenv("LISTEN")
	if listen == "" {
		listen = ":80"
	}
	primary, secondary := mustURL("PRIMARY_URL"), mustURL("SECONDARY_URL")
	log.Printf("listening on %s, primary=%s, mirror writes to %s", listen, primary, secondary)
	log.Fatal(http.ListenAndServe(listen, newHandler(primary, secondary)))
}
