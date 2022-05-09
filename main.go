package main
import "github.com/davecgh/go-spew/spew"
import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func init() {
    _ = spew.Config
}

func main() {
    var script string
    var port int
    flag.StringVar(&script, "s", "./example", "script to be CGIed")
    flag.IntVar(&port, "p", 1234, "port to listen")
    flag.Parse()
    http.ListenAndServe(fmt.Sprintf(":%d", port), NewCGIHandler(script))
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
    fmt.Println(p)
    return CGIHandler{ p }
}


func (c CGIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    cmd := exec.Cmd{}
    cmd.Path = c.script
    cmd.Args = []string{c.script}
    cmd.Args = append(cmd.Args, strings.ToUpper(r.Method))
    toadd := strings.Split(r.URL.Path, "/")
    if r.URL.Path == "/" {
        toadd = []string{}
    }
    if len(toadd) >= 1 && len(toadd[0]) == 0 {
        toadd = toadd[1:]
    }
    cmd.Args = append(cmd.Args, toadd...)
    // spew.Dump(cmd.Args)
    cmd.Env = make([]string, 0, len(r.Header) + 6 + len(os.Environ()))
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
    cmd.Stdout = w
    cmd.Stderr = os.Stderr
    err := cmd.Start()
    if err != nil {
        log.Println(err.Error())

        // w.WriteHeader(500)
        fmt.Fprint(w, err.Error())
        return
    }
    state, err := cmd.Process.Wait()
    if err != nil {
        // w.WriteHeader(500)
        fmt.Fprint(w, err.Error())
        return
    }
    code := state.ExitCode()
    if code == 0 {
        // w.WriteHeader(200)
    } else {
        // w.WriteHeader(code)
    }
}
