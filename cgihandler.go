package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	EnvRemoteAddr       = "REMOTE_ADDR"
	EnvRequestMethod    = "REQUEST_METHOD"
	EnvRequestURI       = "REQUEST_URI"
	EnvServerProtocol   = "SERVER_PROTOCOL"
	EnvServerSoftware   = "SERVER_SOFTWARE"
	EnvScriptFilename   = "SCRIPT_FILENAME"
	EnvServerName       = "SERVER_NAME"
	EnvGatewayInterface = "GATEWAY_INTERFACE"
	EnvQueryPrefix      = "QUERY_%s"
	EnvHeaderPrefix     = "HEADER_%s"

	ServerSoftwareVersion = "ncgi v0.1"
	ServerNameValue       = "ncgi"
	GatewayInterfaceValue = "CGI/1.1"
)

type CGIHandler struct {
	script  string
	bufsize int
}

func NewCGIHandler(script string, bufsize int) http.Handler {
	p, err := exec.LookPath(script)
	if err != nil {
		ReportFatal(err, map[string]interface{}{"script": script})
	}
	p, err = filepath.Abs(p)
	if err != nil {
		ReportFatal(err, map[string]interface{}{"path": p})
	}
	log.Printf("Initializing CGI handler on folder '%s'...", p)
	return CGIHandler{
		script:  p,
		bufsize: bufsize,
	}
}

func (c CGIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cmd := exec.Cmd{}
	cmd.Path = c.script
	cmd.Args = []string{c.script}

	cmd.Args = append(cmd.Args, c.extractPathArgs(r.URL.Path)...)
	cmd.Env = c.buildEnv(r)
	cmd.Stdin = r.Body

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		ReportError(err, nil)
		fmt.Fprint(w, err.Error())
		return
	}
	defer stdout.Close()

	defer func() {
		if cmd.Process != nil {
			err := cmd.Process.Kill()
			if err != nil {
				ReportError(err, nil)
			}
		}
	}()

	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		ReportError(err, map[string]interface{}{"script": c.script})
		fmt.Fprint(w, err.Error())
		return
	}

	c.streamOutput(ctx, w, stdout)
}

func (c CGIHandler) extractPathArgs(path string) []string {
	toadd := strings.Split(path, "/")
	if path == "/" {
		toadd = []string{}
	}
	if len(toadd) >= 1 && len(toadd[0]) == 0 {
		toadd = toadd[1:]
	}
	return toadd
}

func (c CGIHandler) buildEnv(r *http.Request) []string {
	env := make([]string, 0, len(r.Header)+6+len(os.Environ()))
	env = append(env, os.Environ()...)
	env = append(env, fmt.Sprintf("%s=%s", EnvRemoteAddr, r.RemoteAddr))
	env = append(env, fmt.Sprintf("%s=%s", EnvRequestMethod, strings.ToUpper(r.Method)))
	env = append(env, fmt.Sprintf("%s=%s", EnvRequestURI, r.RequestURI))
	env = append(env, fmt.Sprintf("%s=%s", EnvServerProtocol, r.Proto))
	env = append(env, fmt.Sprintf("%s=%s", EnvServerSoftware, ServerSoftwareVersion))
	env = append(env, fmt.Sprintf("%s=%s", EnvScriptFilename, c.script))
	env = append(env, fmt.Sprintf("%s=%s", EnvServerName, ServerNameValue))
	env = append(env, fmt.Sprintf("%s=%s", EnvGatewayInterface, GatewayInterfaceValue))

	for k, v := range r.URL.Query() {
		env = append(env, fmt.Sprintf(EnvQueryPrefix+"=%s", strings.ToUpper(k), strings.Join(v, " ")))
	}

	for k, v := range r.Header {
		env = append(env, fmt.Sprintf(EnvHeaderPrefix+"=%s",
			strings.ReplaceAll(strings.ToUpper(k), "-", "_"),
			strings.Join(v, " "),
		))
	}
	return env
}

func (c CGIHandler) streamOutput(ctx context.Context, w http.ResponseWriter, stdout io.ReadCloser) {
	buf := make([]byte, c.bufsize)
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
				fmt.Fprint(w, err.Error())
				return
			}
			_, err = w.Write(buf[:sz])
			if err != nil {
				ReportError(err, nil)
				fmt.Fprint(w, err.Error())
				return
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}
