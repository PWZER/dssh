package server

import (
	"fmt"
	"log"
	"net"
	"net/http"
)

type FileServerConfig struct {
	Bind    string
	Port    int16
	Root    string
	Encrypt bool
	Daemon  bool
	LogPath string
}

func FileServerStart(cfg *FileServerConfig) error {
	logRequestFunc := func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handler.ServeHTTP(w, r)
			log.Printf("%s %s %s\n", r.RemoteAddr, r.Method, r.URL)
		})
	}

	listener, err := net.Listen("tcp4", fmt.Sprintf("%s:%d", cfg.Bind, cfg.Port))
	if err != nil {
		return err
	}

	log.Printf("server running: http://%s/\n", listener.Addr())
	return http.Serve(listener, logRequestFunc(http.FileServer(http.Dir(cfg.Root))))
}
