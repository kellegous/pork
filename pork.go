package pork

import (
  "bufio"
  "compress/gzip"
  "errors"
  "fmt"
  "io"
  "net"
  "net/http"
  "os"
  "path"
  "path/filepath"
  "runtime"
  "strings"
  "time"
)

type Optimization int

const (
  None Optimization = iota
  Basic
  Advanced
)

type srcType int

const (
  srcOfJsx srcType = iota
  srcOfScss
  srcOfPjs
  srcOfBarrel
  srcOfUnknown
)

type dstType int

const (
  dstOfJs dstType = iota
  dstOfCss
  dstOfUnknown
)

const (
  // src
  barrelFileExtension = ".barrel"
  jsxFileExtension    = ".jsx"
  scssFileExtension   = ".scss"
  pjsFileExtension    = ".pjs"

  // dst
  javaScriptFileExtension = ".js"
  cssFileExtension        = ".css"
)

var PathToCpp = "/usr/bin/gcc"
var PathToJava = "/usr/bin/java"
var PathToRuby = "/usr/bin/ruby"
var PathToNode = "/usr/local/bin/node"

var rootDir string

func pathToJsc() string {
  return filepath.Join(rootDir, "deps/closure/compiler.jar")
}

type command struct {
  args []string
  cwd  string
  env  []string
}

func waitFor(procs []*os.Process) error {
  for _, proc := range procs {
    if proc == nil {
      continue
    }

    s, err := proc.Wait()
    if err != nil {
      return err
    }

    if !s.Success() {
      return errors.New(fmt.Sprintf("exit code: %s", s.Sys()))
    }
  }

  return nil
}

func pipe(c ...*command) (io.ReadCloser, []*os.Process, error) {
  if len(c) == 0 {
    return nil, nil, errors.New("Need commands")
  }

  procs := make([]*os.Process, len(c))
  var r *os.File
  for i, n := 0, len(c); i < n; i++ {
    nr, nw, err := os.Pipe()
    if err != nil {
      if r != nil {
        r.Close()
      }
      return nil, nil, err
    }

    cmd := c[i]
    env := cmd.env
    if cmd.env == nil {
      env = os.Environ()
    }

    p, err := os.StartProcess(
      cmd.args[0],
      cmd.args,
      &os.ProcAttr{
        cmd.cwd,
        env,
        []*os.File{r, nw, os.Stderr},
        nil,
      })
    if err != nil {
      if r != nil {
        r.Close()
      }
      nr.Close()
      nw.Close()
      return nil, nil, err
    }

    // close handles we gave to other processes
    if r != nil {
      r.Close()
    }
    nw.Close()

    procs = append(procs, p)
    r = nr
  }

  return r, procs, nil
}

func prepend(dst []string, args ...string) []string {
  r := make([]string, len(dst)+len(args))
  copy(r, args)
  copy(r[len(args):], dst)
  return r
}

func cppCommand(filename string, includes []string) *command {
  args := []string{
    PathToCpp,
    "-E",
    "-P",
    "-CC",
    "-xc"}

  for _, i := range includes {
    args = append(args, fmt.Sprintf("-I%s", i))
  }

  args = append(args, filename)
  return &command{args, "", nil}
}

func jscCommand(externs []string, jscPath string, level Optimization) *command {
  args := []string{PathToJava, "-jar", jscPath}

  switch level {
  case Basic:
    args = append(args, "--compilation_level", "SIMPLE_OPTIMIZATIONS")
  case Advanced:
    args = append(args, "--compilation_level", "ADVANCED_OPTIMIZATIONS")
  }

  for _, e := range externs {
    args = append(args, "--externs", e)
  }

  return &command{args, "", nil}
}

type Router interface {
  Handle(string, http.Handler)
  HandleFunc(string, func(http.ResponseWriter, *http.Request))
  ServeHTTP(http.ResponseWriter, *http.Request)
}

type Context interface {
  ServeNotFound(http.ResponseWriter, *http.Request)
}

type emptyContext struct{}

func (c *emptyContext) ServeNotFound(w http.ResponseWriter, r *http.Request) {
  http.NotFound(w, r)
}

func ContextFor(w http.ResponseWriter, r *http.Request) Context {
  switch t := w.(type) {
  case *response:
    return t.router
  }
  return &emptyContext{}
}

func NewRouter(logger func(int, *http.Request), notFound http.Handler, headers map[string]string) Router {
  if notFound == nil {
    notFound = http.NotFoundHandler()
  }

  if logger == nil {
    logger = func(status int, r *http.Request) {
    }
  }

  return &router{logger, notFound, headers, http.NewServeMux()}
}

// a response wrapper that provides a couple of additional features:
// (1) a wrapping writer can be interposed (for gzip)
// (2) the status code can be capture for logging
type response struct {
  writer io.Writer
  res    http.ResponseWriter
  router *router
  status int
}

func (r *response) WriteHeader(code int) {
  r.status = code
  r.res.WriteHeader(code)
}

func (r *response) Header() http.Header {
  return r.res.Header()
}

func (r *response) Write(b []byte) (int, error) {
  return r.writer.Write(b)
}

func (r *response) Hijack() (net.Conn, *bufio.ReadWriter, error) {
  return r.res.(http.Hijacker).Hijack()
}

// todo: remove embedded ServerMux and use my trie dispatcher
type router struct {
  logger   func(status int, r *http.Request)
  notFound http.Handler
  headers  map[string]string
  *http.ServeMux
}

func (d *router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  // add any global headers
  h := w.Header()
  for k, v := range d.headers {
    h.Set(k, v)
  }

  res := &response{writer: w, res: w, router: d, status: 200}
  // attempt to interpose a gzip io.Writer
  if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
    g := gzip.NewWriter(w)
    defer g.Close()
    w.Header().Set("Content-Encoding", "gzip")
    res.writer = g
  }

  // dispatch to serving infrastructure
  d.ServeMux.ServeHTTP(res, r)

  // log the request
  d.logger(res.status, r)
}

func (d *router) ServeNotFound(w http.ResponseWriter, r *http.Request) {
  d.notFound.ServeHTTP(w, r)
}

type Handler interface {
  ServeHTTP(w http.ResponseWriter, r *http.Request)
  Productionize(d http.Dir) error
}

type content struct {
  root []http.Dir
  *Config
}

func pathToThisFile() string {
  _, file, _, _ := runtime.Caller(0)
  return file
}

func Root() string {
  return rootDir
}

func init() {
  rootDir = filepath.Dir(pathToThisFile())
}

type HostRedirectHandler string

func (h HostRedirectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  http.Redirect(w, r,
    fmt.Sprintf("http://%s%s", string(h), r.RequestURI),
    http.StatusMovedPermanently)
}

func ErrorFileHandler(path string, code int) http.Handler {
  if _, err := os.Stat(path); err != nil {
    panic(err)
  }
  return &errorFileHandler{path, code}
}

type errorFileHandler struct {
  path string
  code int
}

func (h *errorFileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  w.Header().Set("Content-Type", "text/html; charset=utf-8")
  w.WriteHeader(h.code)
  f, err := os.Open(h.path)
  if err != nil {
    panic(err)
  }
  defer f.Close()

  _, err = io.Copy(w, f)
  if err != nil {
    panic(err)
  }
}

// returns true if content should be delivered
func ServeContentModificationTime(w http.ResponseWriter, r *http.Request, t time.Time) bool {
  if t.IsZero() {
    return true
  }

  // The Date-Modified header truncates sub-second precision, so
  // use mtime < t+1s instead of mtime <= t to check for unmodified.
  if ht, err := time.Parse(http.TimeFormat, r.Header.Get("If-Modified-Since")); err == nil && t.Before(ht.Add(1*time.Second)) {
    w.WriteHeader(http.StatusNotModified)
    return false
  }
  w.Header().Set("Last-Modified", t.UTC().Format(http.TimeFormat))
  return true
}

func FileHandler(path string) http.Handler {
  if _, err := os.Stat(path); err != nil {
    panic(err)
  }
  return fileHandler(path)
}

type fileHandler string

func (h fileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  http.ServeFile(w, r, string(h))
}

type Config struct {
  Level        Optimization
  PjsIncludes  []string
  PjsExterns   []string
  JsxIncludes  []string
  JsxExterns   []string
  ScssIncludes []string
}

func NewConfig(level Optimization) *Config {
  return &Config{
    Level:       level,
    PjsIncludes: []string{filepath.Join(rootDir, "js")},
  }
}

func Content(c *Config, d ...http.Dir) Handler {
  return &content{d, c}
}

func expandPath(fs http.Dir, name string) string {
  return filepath.Join(string(fs), filepath.FromSlash(path.Clean("/"+name)))
}

type typeFound int

const (
  foundNothing typeFound = iota
  foundFile
  foundDirectory
)

func findFile(d []http.Dir, name string) (string, typeFound) {
  for i, n := 0, len(d); i < n; i++ {
    target := filepath.Join(string(d[i]), filepath.FromSlash(path.Clean("/"+name)))

    // if the file doesn't exist, move along
    s, err := os.Stat(target)
    if err != nil {
      continue
    }

    // if it's a file, return that
    if !s.IsDir() {
      return target, foundFile
    }

    // if it's a dir, check for an index
    target = filepath.Join(target, "index.html")
    if _, err := os.Stat(target); err == nil {
      return target, foundDirectory
    }
  }
  return "", foundNothing
}

func changeTypeOfFile(path, from, to string) string {
  // assert path ends with from
  return path[0:len(path)-len(from)] + to
}

func typeOfSrc(filename string) srcType {
  ext := filepath.Ext(filename)
  switch ext {
  case jsxFileExtension:
    return srcOfJsx
  case pjsFileExtension:
    return srcOfPjs
  case scssFileExtension:
    return srcOfScss
  case barrelFileExtension:
    return srcOfBarrel
  }
  return srcOfUnknown
}

func typeOfDst(filename string) dstType {
  switch filepath.Ext(filename) {
  case cssFileExtension:
    return dstOfCss
  case javaScriptFileExtension:
    return dstOfJs
  }
  return dstOfUnknown
}

func ServeContent(c Context, w http.ResponseWriter, r *http.Request, cfg *Config, d ...http.Dir) {
  path := r.URL.Path

  // if the file exists, just serve it.
  if target, found := findFile(d, path); found != foundNothing {
    // if this is a directory without a trailing /, we need to normalize.
    if found == foundDirectory && path[len(path)-1] != '/' {
      http.Redirect(w, r, path+"/", http.StatusMovedPermanently)
      return
    }
    http.ServeFile(w, r, target)
    return
  }

  switch typeOfDst(path) {
  case dstOfJs:
    // try to answer with jsx
    jsxSrc, found := findFile(d, changeTypeOfFile(path, javaScriptFileExtension, jsxFileExtension))
    if found == foundFile {
      // serve jsx
      w.Header().Set("Content-Type", "text/javascript")
      if err := CompileJsx(cfg, jsxSrc, w); err != nil {
        panic(err)
      }
      return
    }

    // try to answer with pjs
    pjsSrc, found := findFile(d, changeTypeOfFile(path, javaScriptFileExtension, pjsFileExtension))
    if found == foundFile {
      w.Header().Set("Content-Type", "text/javascript")
      if err := CompilePjs(cfg, pjsSrc, w); err != nil {
        panic(err)
      }
      return
    }

    // try to answer with a barrel
    brlSrc, found := findFile(d, changeTypeOfFile(path, javaScriptFileExtension, barrelFileExtension))
    if found == foundFile {
      w.Header().Set("Content-Type", "text/javascript")
      if err := CompileBarrel(cfg, brlSrc, w); err != nil {
        panic(err)
      }
      return
    }
    c.ServeNotFound(w, r)
  case dstOfCss:
    cssSrc, found := findFile(d, changeTypeOfFile(path, cssFileExtension, scssFileExtension))
    if found == foundFile {
      w.Header().Set("Content-Type", "text/css")
      if err := CompileScss(cfg, cssSrc, w); err != nil {
        panic(err)
      }
      return
    }
  default:
    c.ServeNotFound(w, r)
  }
}

func (h *content) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  ServeContent(ContextFor(w, r), w, r, h.Config, h.root...)
}

func rebasePath(src, dst, filename string) (string, error) {
  target, err := filepath.Rel(src, filename)
  if err != nil {
    return "", err
  }
  return filepath.Join(dst, target), nil
}

func compileToFile(c *Config, src, dst string, fn func(*Config, string, io.Writer) error) error {
  // ensure we have all the directories we need
  dir := filepath.Dir(dst)
  if _, err := os.Stat(dir); err != nil {
    if err := os.MkdirAll(dir, 0777); err != nil {
      return err
    }
  }

  file, err := os.Create(dst)
  if err != nil {
    return err
  }
  defer file.Close()

  if err := fn(c, src, file); err != nil {
    return err
  }

  return nil
}

func (h *content) Productionize(d http.Dir) error {
  dst := string(d)
  if _, err := os.Stat(dst); err != nil {
    if err := os.MkdirAll(dst, 0777); err != nil {
      return err
    }
  }

  for _, root := range h.root {
    src := string(root)
    filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
      switch typeOfSrc(path) {
      case srcOfJsx:
        target, err := rebasePath(src, dst,
          changeTypeOfFile(path, jsxFileExtension, javaScriptFileExtension))
        if err != nil {
          return err
        }

        if err := compileToFile(h.Config, path, target, CompileJsx); err != nil {
          return err
        }
      case srcOfPjs:
        target, err := rebasePath(src, dst,
          changeTypeOfFile(path, pjsFileExtension, javaScriptFileExtension))
        if err != nil {
          return err
        }

        if err := compileToFile(h.Config, path, target, CompilePjs); err != nil {
          return err
        }
      case srcOfBarrel:
        target, err := rebasePath(src, dst,
          changeTypeOfFile(path, barrelFileExtension, javaScriptFileExtension))
        if err != nil {
          return err
        }

        if err := compileToFile(h.Config, path, target, CompileBarrel); err != nil {
          return err
        }
      case srcOfScss:
        // todo: there is an issue here in that I will compile things
        // that are only intended to be included.
        target, err := rebasePath(src, dst,
          changeTypeOfFile(path, scssFileExtension, cssFileExtension))
        if err != nil {
          return err
        }

        if err := compileToFile(h.Config, path, target, CompileScss); err != nil {
          return err
        }
      }
      return nil
    })
  }

  // todo: should be first
  h.root = append(h.root, d)
  return nil
}
