package pork

import (
  "bufio"
  "compress/gzip"
  "encoding/json"
  "errors"
  "fmt"
  "net"
  "net/http"
  "io"
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

type fileType int
const (
  javascript fileType = iota
  porkBundle
  css
  scss
  unknown
)

const (
  porkBundleFileExtension = ".pork"
  javaScriptFileExtension = ".js"
  cssFileExtension = ".css"
  sassFileExtension = ".scss"
)

var PathToCpp = "/usr/bin/cpp"
var PathToJava = "/usr/bin/java"
var PathToRuby = "/usr/bin/ruby"

var rootDir string

func pathToJsc() string {
  return filepath.Join(rootDir, "deps/closure/compiler.jar")
}

func pathToSass() string {
  return filepath.Join(rootDir, "deps/sass/bin/sass")
}

func waitFor(procs ...*os.Process) error {
  for _, proc := range procs {
    if proc == nil {
      continue
    }

    s, err := proc.Wait(0)
    if err != nil {
      return err
    }

    if sc := s.WaitStatus.ExitStatus(); sc != 0 {
      return errors.New(fmt.Sprintf("exit code: %d", sc))      
    }
  }

  return nil
}

func cpp(filename string, w *os.File) (*os.Process, error) {
  cppArgs := []string{
    PathToCpp,
    "-P",
    "-CC",
    fmt.Sprintf("-I%s", filepath.Join(rootDir, "src")),
    filename}
  return os.StartProcess(cppArgs[0],
    cppArgs,
    &os.ProcAttr{
      "",
      os.Environ(),
      []*os.File{nil, w, os.Stderr},
      nil})
}

func jsc(r *os.File, w *os.File, externs, jscPath string, level Optimization) (*os.Process, error) {
  jvmArgs := []string{PathToJava, "-jar", jscPath}
  if level == Advanced {
    jvmArgs = append(jvmArgs, "--compilation_level", "ADVANCED_OPTIMIZATIONS")
  }

  if externs != "" {
    jvmArgs = append(jvmArgs, "--externs", externs)
  }

  return os.StartProcess(jvmArgs[0],
    jvmArgs,
    &os.ProcAttr{
      "",
      os.Environ(),
      []*os.File{r, w, os.Stderr},
      nil})
}

func sass(filename string, w *os.File, sassPath string) (*os.Process, error) {
  sassArgs := []string{
      PathToRuby,
      sassPath,
      "--no-cache",
      filename}
  return os.StartProcess(sassArgs[0],
    sassArgs,
    &os.ProcAttr{
      "",
      os.Environ(),
      []*os.File{nil, w, os.Stderr},
      nil})
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
  res http.ResponseWriter
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
  logger func(status int, r *http.Request)
  notFound http.Handler
  headers map[string]string
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
  level Optimization
}

func pathToThisFile() string {
  _, file, _, _ := runtime.Caller(0)
  return file
}

func init() {
  rootDir = filepath.Dir(filepath.Dir(filepath.Dir(pathToThisFile())))
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
   if ht, err := time.Parse(http.TimeFormat, r.Header.Get("If-Modified-Since")); err == nil && t.Before(ht.Add(1 * time.Second)) {
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

func Content(level Optimization, d ...http.Dir) Handler {
  return &content{d, level}
}

func expandPath(fs http.Dir, name string) string {
  return filepath.Join(string(fs), filepath.FromSlash(path.Clean("/" + name)))
}

func findFile(d []http.Dir, name string) (string, bool) {
  for i, n := 0, len(d); i < n; i++ {
    target := filepath.Join(string(d[i]), filepath.FromSlash(path.Clean("/" + name)))

    // if the file doesn't exist, move along
    s, err := os.Stat(target)
    if err != nil {
      continue
    }

    // if it's a file, return that
    if !s.IsDir() {
      return target, true
    }

    // if it's a dir, check for an index
    target = filepath.Join(target, "index.html")
    if _, err := os.Stat(target); err == nil {
      return target, true
    }
  }
  return "", false
}

func changeTypeOfFile(path, from, to string) string {
  // assert path ends with from
  return path[0 : len(path) - len(from)] + to
}

func typeOfFile(filename string) fileType {
  if strings.HasSuffix(filename, porkBundleFileExtension) {
    return porkBundle
  }
  if strings.HasSuffix(filename, javaScriptFileExtension) {
    return javascript
  }
  if strings.HasSuffix(filename, cssFileExtension) {
    return css
  }
  if strings.HasSuffix(filename, sassFileExtension) {
    return scss
  }
  return unknown
}

func ServeContent(c Context, w http.ResponseWriter, r *http.Request, level Optimization, d ...http.Dir) {
  path := r.URL.Path

  // if the file exists, just serve it.
  if target, found := findFile(d, path); found {
    http.ServeFile(w, r, target)
    return
  }

  switch typeOfFile(path) {
  case javascript:
    source, found := findFile(d, changeTypeOfFile(path, javaScriptFileExtension, porkBundleFileExtension))
    if !found {
      c.ServeNotFound(w, r)
      return
    }
    w.Header().Set("Content-Type", "text/javascript")
    err := CompileBundle(source, w, level)
    if err != nil {
      // todo: send to ServeSiteError
      panic(err)
    }
  case css:
    source, found := findFile(d, changeTypeOfFile(path, cssFileExtension, sassFileExtension))
    if !found {
      c.ServeNotFound(w, r)
      return
    }
    w.Header().Set("Content-Type", "text/css")
    err := CompileCss(source, w)
    if err != nil {
      // todo: send to ServeSiteError
      panic(err)
    }
  default:
    c.ServeNotFound(w, r)
  }
}

func (h *content) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  ServeContent(ContextFor(w, r), w, r, h.level, h.root...)
}

func rebasePath(src, dst, filename string) (string, error) {
  target, err := filepath.Rel(src, filename)
  if err != nil {
    return "", err
  }
  return filepath.Join(dst, target), nil
}

func createFile(path string) (*os.File, error) {
  dir, _ := filepath.Split(path)
  if _, err := os.Stat(dir); err != nil {
    if err := os.MkdirAll(dir, 0777); err != nil {
      return nil, err
    }
  }
  return os.Create(path)
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
      switch typeOfFile(path) {
      case porkBundle:
        target, err := rebasePath(src, dst,
          changeTypeOfFile(path, porkBundleFileExtension, javaScriptFileExtension))
        if err != nil {
          return err
        }

        file, err := createFile(target)
        if err != nil {
          return err
        }
        defer file.Close()

        if err := CompileBundle(path, file, h.level); err != nil {
          return err
        }
      case scss:
        // todo: there is an issue here in that I will compile things
        // that are only intended to be included.
        target, err := rebasePath(src, dst,
          changeTypeOfFile(path, sassFileExtension, cssFileExtension))
        if err != nil {
          return err
        }

        file, err := createFile(target)
        if err != nil {
          return err
        }
        defer file.Close()

        if err := CompileCss(path, file); err != nil {
          return err
        }
      }
      return nil
    })
  }

  h.root = append(h.root, d)
  return nil
}

func CompileCss(filename string, w io.Writer) error {
  rp, wp, err := os.Pipe()
  if err != nil {
    return err
  }
  defer rp.Close()
  defer wp.Close()

  p, err := sass(filename, wp, pathToSass())
  if err != nil {
    return err
  }
  wp.Close()

  err = waitFor(p)
  if err != nil {
    return err
  }

  _, err = io.Copy(w, rp)
  if err != nil {
    return err
  }

  return nil
}


type bundle struct {
  Externs string
  Units []*unit
}

type unit struct {
  File string
  LevelStr string `json:"Level"`
}

func (u *unit) Level() Optimization {
  switch u.LevelStr {
  case "Basic":
    return Basic
  case "None":
    return None
  }
  return Advanced
}

func loadBundle(r io.Reader) (*bundle, error) {
  b := &bundle{}
  err := json.NewDecoder(r).Decode(&b)
  if err != nil {
    return nil, err
  }
  return b, nil
}

func minLevel(a Optimization, b Optimization) Optimization {
  if a < b {
    return a
  }
  return b
}

func CompileBundle(filename string, w io.Writer, level Optimization) error {
  dir, _ := filepath.Split(filename)

  file, err := os.Open(filename)
  if err != nil {
    return err
  }
  defer file.Close()

  bundle, err := loadBundle(file)
  if err != nil {
    return err
  }

  externs := ""
  if bundle.Externs != "" {
    externs = filepath.Join(dir, bundle.Externs)
  }

  for _, u := range bundle.Units {
    err := compileJs(filepath.Join(dir, u.File),
      externs,
      w,
      minLevel(u.Level(),
      level))
    if err != nil {
      return err
    }
  }

  return nil
}

// todo: make this private.
func compileJs(filename, externs string, w io.Writer, level Optimization) error {
  // output pipe
  orp, owp, err := os.Pipe()
  if err != nil {
    return err
  }
  defer orp.Close()
  defer owp.Close()

  var cp, jp *os.Process
  switch level {
  case None:
    cp, err = cpp(filename, owp)
    if err != nil {
      return err
    }
    owp.Close()
  case Basic, Advanced:
    irp, iwp, err := os.Pipe()
    if err != nil {
      return err
    }
    defer irp.Close()
    defer iwp.Close()

    cp, err = cpp(filename, iwp)
    if err != nil {
      return err
    }

    iwp.Close()

    jp, err = jsc(irp, owp, externs, pathToJsc(), level)
    if err != nil {
      return err
    }

    irp.Close()
    owp.Close()
  }

  _, err = io.Copy(w, orp)
  if err != nil {
    return err
  }

  err = waitFor(cp, jp)
  if err != nil {
    return err
  }

  return nil
}
