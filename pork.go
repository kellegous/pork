package pork

import (
  "bufio"
  "compress/gzip"
  "encoding/json"
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
  Super
)

type fileType int

const (
  javascript fileType = iota
  porkBundle
  jsx
  css
  scss
  unknown
)

const (
  porkBundleFileExtension = ".pork"
  jsxFileExtension        = ".jsx"
  javaScriptFileExtension = ".js"
  cssFileExtension        = ".css"
  sassFileExtension       = ".scss"
)

var PathToCpp = "/usr/bin/gcc"
var PathToJava = "/usr/bin/java"
var PathToRuby = "/usr/bin/ruby"
var PathToNode = "/usr/local/bin/node"

var rootDir string

func pathToJsc() string {
  return filepath.Join(rootDir, "deps/closure/compiler.jar")
}

func pathToSass() string {
  return filepath.Join(rootDir, "sass")
}

func pathToJsx() string {
  return filepath.Join(rootDir, "deps/jsx/bin/jsx")
}

type command struct {
  args []string
  cwd  string
  env  []string
}

func pipe(c ...*command) (io.ReadCloser, error) {
  if len(c) == 0 {
    return nil, errors.New("Need commands")
  }

  var r *os.File
  for i, n := 0, len(c); i < n; i++ {
    nr, nw, err := os.Pipe()
    if err != nil {
      if r != nil {
        r.Close()
      }
      return nil, err
    }

    cmd := c[i]
    env := cmd.env
    if cmd.env == nil {
      env = os.Environ()
    }

    _, err = os.StartProcess(
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
      return nil, err
    }

    // close handles we gave to other processes
    if r != nil {
      r.Close()
    }
    nw.Close()

    r = nr
  }

  return r, nil
}

func cppCommand(filename string, includes []string) *command {
  args := []string{
    PathToCpp,
    "-E",
    "-P",
    "-CC",
    "-xc",
    fmt.Sprintf("-I%s", filepath.Join(rootDir, "js"))}

  for _, i := range includes {
    args = append(args, fmt.Sprintf("-I%s", i))
  }

  args = append(args, filename)
  return &command{args, "", nil}
}

func jscCommand(externs []string, jscPath string, level Optimization) *command {
  args := []string{PathToJava, "-jar", jscPath}
  switch level {
  case Advanced:
    args = append(args, "--compilation_level", "ADVANCED_OPTIMIZATIONS")
  case Super:
    args = append(args, "--compilation_level", "SUPER_OPTIMIZATIONS")
  }

  for _, e := range externs {
    args = append(args, "--externs", e)
  }

  return &command{args, "", nil}
}

func sassCommand(filename string, sassPath string, level Optimization) *command {
  args := []string{
    PathToRuby,
    sassPath,
    "--no-cache",
    "--trace"}

  switch level {
  case Basic:
    args = append(args, "--style", "compact")
  case Advanced:
    args = append(args, "--style", "compressed")
  }

  args = append(args, filename)
  return &command{args, "", nil}
}

func jsxCommand(filename, jsxPath string) *command {
  return &command{
    []string{
      PathToNode,
      jsxPath,
      filename,
    }, "", nil}
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
  root  []http.Dir
  level Optimization
}

func pathToThisFile() string {
  _, file, _, _ := runtime.Caller(0)
  return file
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

func Content(level Optimization, d ...http.Dir) Handler {
  return &content{d, level}
}

func expandPath(fs http.Dir, name string) string {
  return filepath.Join(string(fs), filepath.FromSlash(path.Clean("/"+name)))
}

func findFile(d []http.Dir, name string) (string, bool) {
  for i, n := 0, len(d); i < n; i++ {
    target := filepath.Join(string(d[i]), filepath.FromSlash(path.Clean("/"+name)))

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
  return path[0:len(path)-len(from)] + to
}

func typeOfFile(filename string) fileType {
  if strings.HasSuffix(filename, porkBundleFileExtension) {
    return porkBundle
  }
  if strings.HasSuffix(filename, jsxFileExtension) {
    return jsx
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
    jsxSrc, found := findFile(d, changeTypeOfFile(path, javaScriptFileExtension, jsxFileExtension))
    if found {
      // serve jsx
      w.Header().Set("Content-Type", "text/javascript")
      if err := CompileJsx(jsxSrc, w, level); err != nil {
        panic(err)
      }
      return
    }

    prkSrc, found := findFile(d, changeTypeOfFile(path, javaScriptFileExtension, porkBundleFileExtension))
    if found {
      // serve porkjs
      w.Header().Set("Content-Type", "text/javascript")
      if err := CompileBundle(prkSrc, w, level); err != nil {
        panic(err)
      }
      return
    }

    c.ServeNotFound(w, r)
  case css:
    source, found := findFile(d, changeTypeOfFile(path, cssFileExtension, sassFileExtension))
    if !found {
      c.ServeNotFound(w, r)
      return
    }
    w.Header().Set("Content-Type", "text/css")
    err := CompileCss(source, w, level)
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
      case jsx:
        target, err := rebasePath(src, dst,
          changeTypeOfFile(path, jsxFileExtension, javaScriptFileExtension))
        if err != nil {
          return err
        }

        file, err := createFile(target)
        if err != nil {
          return err
        }
        defer file.Close()

        if err := CompileJsx(path, file, h.level); err != nil {
          return err
        }
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

        if err := CompileCss(path, file, h.level); err != nil {
          return err
        }
      }
      return nil
    })
  }

  h.root = append(h.root, d)
  return nil
}

func CompileCss(filename string, w io.Writer, level Optimization) error {
  r, err := pipe(sassCommand(filename, pathToSass(), level))
  if err != nil {
    return err
  }
  defer r.Close()

  _, err = io.Copy(w, r)
  if err != nil {
    return err
  }

  return nil
}

type bundle struct {
  Externs  []string
  Includes []string
  Units    []*unit
}

type unit struct {
  File     string
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

func CompileJsx(filename string, w io.Writer, level Optimization) error {
  var r io.ReadCloser
  var err error

  switch level {
  case None:
    r, err = pipe(jsxCommand(filename, pathToJsx()))
    if err != nil {
      return err
    }
    defer r.Close()
  case Basic, Advanced:
    r, err = pipe(
      jsxCommand(filename, pathToJsx()),
      jscCommand([]string{}, pathToJsc(), level))
    if err != nil {
      return err
    }
    defer r.Close()
  }

  _, err = io.Copy(w, r)
  if err != nil {
    return err
  }

  // this is a hack for now.
  _, err = fmt.Fprintf(w, "JSX.require(\"%s\")._Main.main$([]);\n", filename)
  if err != nil {
    return err
  }

  return nil
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

  externs := []string{}
  if bundle.Externs != nil {
    for _, e := range bundle.Externs {
      externs = append(externs, filepath.Join(dir, e))
    }
  }

  includes := []string{}
  if bundle.Includes != nil {
    for _, i := range bundle.Includes {
      includes = append(includes, filepath.Join(dir, i))
    }
  }

  for _, u := range bundle.Units {
    err := compileJs(filepath.Join(dir, u.File),
      externs,
      includes,
      w,
      minLevel(u.Level(),
        level))
    if err != nil {
      return err
    }
  }

  return nil
}

func compileJs(filename string, externs []string, includes []string, w io.Writer, level Optimization) error {

  var r io.ReadCloser
  var err error

  switch level {
  case None:
    r, err = pipe(cppCommand(filename, includes))
    if err != nil {
      return err
    }
    defer r.Close()
  case Basic, Advanced:
    r, err = pipe(
      cppCommand(filename, includes),
      jscCommand(externs, pathToJsc(), level))
    if err != nil {
      return err
    }
    defer r.Close()
  }

  _, err = io.Copy(w, r)
  if err != nil {
    return err
  }

  return nil
}
