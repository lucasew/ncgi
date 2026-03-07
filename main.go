package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/davecgh/go-spew/spew"
	"github.com/phayes/freeport"
)

func init() {
	_ = spew.Config
}

const DefaultBufferSize = 64 * 1024

type Server struct {
	httpServer *http.Server
	port       int
}

func NewServer(port int, handler http.Handler) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: handler,
		},
		port: port,
	}
}

func (s *Server) Start(ctx context.Context) {
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			ReportFatal(err, nil)
		}
	}()
}

func (s *Server) Stop(ctx context.Context) {
	if err := s.httpServer.Shutdown(ctx); err != nil {
		ReportError(err, nil)
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var script string
	var port int
	var bufsize int

	flag.StringVar(&script, "s", "./example", "script to be CGIed")
	flag.IntVar(&port, "p", 0, "port to listen")
	flag.IntVar(&bufsize, "b", DefaultBufferSize, "buffer size")
	flag.Parse()

	if port == 0 {
		port = freeport.GetPort()
	}
	log.Printf("Listening on port %d...", port)

	handler := NewCGIHandler(script, bufsize)
	srv := NewServer(port, handler)

	srv.Start(ctx)
	defer srv.Stop(ctx)

	if err := handleSubprocess(ctx, port, flag.Args()...); err != nil {
		ReportFatal(err, nil)
	}
}
