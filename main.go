package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/phayes/freeport"
)

func init() {
	_ = spew.Config
}

var bufsize int
var script string
var port int

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	flag.StringVar(&script, "s", "./example", "script to be CGIed")
	flag.IntVar(&port, "p", 0, "port to listen")
	flag.IntVar(&bufsize, "b", 64*1024, "buffer size")
	flag.Parse()
	if port == 0 {
		port = freeport.GetPort()
	}
	log.Printf("Listening on port %d...", port)
	server := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: NewCGIHandler(script)}
	defer func() {
		err := server.Shutdown(ctx)
		if err != nil {
			ReportError(err, nil)
		}
	}()
	go func() {
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			ReportFatal(err, nil)
		}
	}()

	err := handleSubprocess(ctx, flag.Args()...)
	if err != nil {
		ReportFatal(err, nil)
	}
}

func handleSubprocess(ctx context.Context, args ...string) error {
	var err error
	if len(args) == 0 {
		for {
			time.Sleep(1 * time.Second)
		}
	}
	args[0], err = exec.LookPath(args[0])
	if err != nil {
		return err
	}
	for i := range args {
		args[i] = strings.ReplaceAll(args[i], "%PORT%", fmt.Sprintf("%d", port))
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	time.Sleep(1 * time.Second)
	return cmd.Run()
}

type CGIHandler struct {
	script string
}

func NewCGIHandler(script string) http.Handler {
	p, err := exec.LookPath(script)
	if err != nil {
		ReportFatal(err, map[string]interface{}{"script": script})
	}
	p, err = filepath.Abs(p)
	if err != nil {
		ReportFatal(err, map[string]interface{}{"path": p})
	}
	log.Printf("Initializing CGI handler on folder '%s'...", p)
	return CGIHandler{p}
}

func (c CGIHandler) buildArgs(r *http.Request) ([]string, error) {
	args := []string{c.script}
	toadd := strings.Split(r.URL.Path, "/")
	if r.URL.Path == "/" {
		toadd = []string{}
	}
	if len(toadd) >= 1 && len(toadd[0]) == 0 {
		toadd = toadd[1:]
	}
	for _, segment := range toadd {
		if strings.HasPrefix(segment, "-") {
			return nil, fmt.Errorf("invalid path segment: %s", segment)
		}
	}
	return append(args, toadd...), nil
}

func (c CGIHandler) buildEnv(r *http.Request) []string {
	env := make([]string, 0, len(r.Header)+6+len(os.Environ()))
	env = append(env, os.Environ()...)
	env = append(env, fmt.Sprintf("REMOTE_ADDR=%s", r.RemoteAddr))
	env = append(env, fmt.Sprintf("REQUEST_METHOD=%s", strings.ToUpper(r.Method)))
	env = append(env, fmt.Sprintf("REQUEST_URI=%s", r.RequestURI))
	env = append(env, fmt.Sprintf("SERVER_PROTOCOL=%s", r.Proto))
	env = append(env, "SERVER_SOFTWARE=ncgi v0.1")
	env = append(env, fmt.Sprintf("SCRIPT_FILENAME=%s", c.script))
	env = append(env, "SERVER_NAME=ncgi")
	env = append(env, "GATEWAY_INTERFACE=CGI/1.1")
	for k, v := range r.URL.Query() {
		env = append(env, fmt.Sprintf("QUERY_%s=%s", strings.ToUpper(k), strings.Join(v, " ")))
	}
	for k, v := range r.Header {
		env = append(env, fmt.Sprintf("HEADER_%s=%s",
			strings.ReplaceAll(strings.ToUpper(k), "-", "_"),
			strings.Join(v, " "),
		))
	}
	return env
}

func (c CGIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cmd := exec.Cmd{}
	cmd.Path = c.script

	args, err := c.buildArgs(r)
	if err != nil {
		ReportError(err, map[string]interface{}{"path": r.URL.Path})
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	cmd.Args = args
	cmd.Env = c.buildEnv(r)

	cmd.Stdin = r.Body
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		ReportError(err, nil)
		_, errWrite := fmt.Fprint(w, err.Error())
		if errWrite != nil {
			ReportError(errWrite, nil)
		}
		return
	}
	defer func() {
		if err := stdout.Close(); err != nil {
			ReportError(err, nil)
		}
	}()
	defer func() {
		if cmd.Process != nil {
			err := cmd.Process.Kill()
			if err != nil {
				ReportError(err, nil)
			}
		}
	}()
	// cmd.Stdout = w
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		ReportError(err, map[string]interface{}{"script": c.script})

		// w.WriteHeader(500)
		_, errWrite := fmt.Fprint(w, err.Error())
		if errWrite != nil {
			ReportError(errWrite, nil)
		}
		return
	}
	buf := make([]byte, bufsize)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			sz, err := stdout.Read(buf)
			if err == io.EOF {
				return
			}
			if err != nil {
				ReportError(err, nil)
				_, errWrite := fmt.Fprint(w, err.Error())
				if errWrite != nil {
					ReportError(errWrite, nil)
				}
				return
			}
			_, err = w.Write(buf[:sz])
			if err != nil {
				ReportError(err, nil)
				_, errWrite := fmt.Fprint(w, err.Error())
				if errWrite != nil {
					ReportError(errWrite, nil)
				}
				return
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}
