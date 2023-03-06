package main

import (
	"context"
	"flag"
	"fmt"
	"time"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
	defer server.Shutdown(ctx)
	go server.ListenAndServe()

	err := handleSubprocess(ctx, flag.Args()...)
	if err != nil {
		panic(err)
	}
}

func handleSubprocess(ctx context.Context, args ...string) error {
	var err error
	if len(args) == 0 {
		for {
			time.Sleep(1*time.Second)
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
	time.Sleep(1*time.Second)
	return cmd.Run()
}

type CGIHandler struct {
	script string
}

func NewCGIHandler(script string) http.Handler {
	p, err := exec.LookPath(script)
	if err != nil {
		panic(err)
	}
	p, err = filepath.Abs(p)
	if err != nil {
		panic(err)
	}
	log.Printf("Initializing CGI handler on folder '%s'...", p)
	return CGIHandler{p}
}

func (c CGIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cmd := exec.Cmd{}
	cmd.Path = c.script
	cmd.Args = []string{c.script}
	// cmd.Args = append(cmd.Args, strings.ToUpper(r.Method))
	toadd := strings.Split(r.URL.Path, "/")
	if r.URL.Path == "/" {
		toadd = []string{}
	}
	if len(toadd) >= 1 && len(toadd[0]) == 0 {
		toadd = toadd[1:]
	}
	cmd.Args = append(cmd.Args, toadd...)
	cmd.Env = make([]string, 0, len(r.Header)+6+len(os.Environ()))
	for _, env := range os.Environ() {
		cmd.Env = append(cmd.Env, env)
	}
	cmd.Env = append(cmd.Env, fmt.Sprintf("REMOTE_ADDR=%s", r.RemoteAddr))
	cmd.Env = append(cmd.Env, fmt.Sprintf("REQUEST_METHOD=%s", strings.ToUpper(r.Method)))
	cmd.Env = append(cmd.Env, fmt.Sprintf("REQUEST_URI=%s", r.RequestURI))
	cmd.Env = append(cmd.Env, fmt.Sprintf("SERVER_PROTOCOL=%s", r.Proto))
	cmd.Env = append(cmd.Env, "SERVER_SOFTWARE=ncgi v0.1")
	cmd.Env = append(cmd.Env, fmt.Sprintf("SCRIPT_FILENAME=%s", c.script))
	cmd.Env = append(cmd.Env, "SERVER_NAME=ncgi")
	cmd.Env = append(cmd.Env, "GATEWAY_INTERFACE=CGI/1.1")
	for k, v := range r.URL.Query() {
		cmd.Env = append(cmd.Env, fmt.Sprintf("QUERY_%s=%s", strings.ToUpper(k), strings.Join(v, " ")))
	}
	for k, v := range r.Header {
		cmd.Env = append(cmd.Env, fmt.Sprintf("HEADER_%s=%s",
			strings.ReplaceAll(strings.ToUpper(k), "-", "_"),
			strings.Join(v, " "),
		))
	}
	cmd.Stdin = r.Body
	stdout, err := cmd.StdoutPipe()
	defer stdout.Close()
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()
	// cmd.Stdout = w
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		log.Println(err.Error())

		// w.WriteHeader(500)
		fmt.Fprint(w, err.Error())
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
				fmt.Fprint(w, err.Error())
				return
			}
			_, err = w.Write(buf[:sz])
			if err != nil {
				fmt.Fprint(w, err.Error())
				return
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}
