package sse

import (
	"net/http"
	"time"
)

func (s *Streamer) SetHTTPHeaders(rw http.ResponseWriter) {
	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Connection", "keep-alive")
	if s.options.NoXAccelBuffering {
		rw.Header().Set("X-Accel-Buffering", "no")
	}
}

func (s *Streamer) Dispatcher(client *Client, rw http.ResponseWriter, r *http.Request) {
	fl, ok := rw.(http.Flusher)
	if !ok {
		rw.WriteHeader(http.StatusNotImplemented) // unsupported
		return
	}

	cn := r.Context().Done()
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()

	for {
		select {
		case <-cn: // disconnected from client
			s.removeClient(client)
			// consume all buffer
			for {
				select {
				case <-client.send:
				default:
					return
				}
			}
		case payload, ok := <-client.send: // send event
			if ok {
				if len(payload.event) > 0 {
					rw.Write([]byte("event:"))
					rw.Write([]byte(payload.event))
					rw.Write([]byte("\n"))
				}
				for _, v := range payload.data {
					rw.Write([]byte("data:"))
					rw.Write([]byte(v))
					rw.Write([]byte("\n"))
				}
				rw.Write([]byte("\n"))
				fl.Flush()
			} else {
				return // streamer stopped
			}
		case <-t.C: // prevent timeout
			rw.Write([]byte(":\n\n"))
			fl.Flush()
		}
	}
}

func (s *Streamer) Handler(rw http.ResponseWriter, r *http.Request) {
	fl, ok := rw.(http.Flusher)
	if !ok {
		rw.WriteHeader(http.StatusNotImplemented) // unsupported
		return
	}

	var (
		err     error
		userKey string
	)
	if s.options.UserAuthenticator != nil {
		userKey, err = s.options.UserAuthenticator(r)
		if err != nil {

		}
	}

	client, err := s.NewClient(userKey)
	if err != nil {
		rw.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	// set headers
	s.SetHTTPHeaders(rw)

	rw.WriteHeader(http.StatusOK)
	fl.Flush()

	s.Dispatcher(client, rw, r)
}
