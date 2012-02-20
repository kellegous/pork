package pork

import (
  "compress/gzip"
  "errors"
  "fmt"
  "net/http"
  "io"
  "os"
  "path"
  "path/filepath"
  "runtime"
  "strings"
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
  porkscript
  css
  scss
  unknown
)

const (
  porkScriptFileExtension = ".pork.js"
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

func jsc(r *os.File, w *os.File, jscPath string, level Optimization) (*os.Process, error) {
  jvmArgs := []string{PathToJava, "-jar", jscPath}
  if level == Advanced {
    jvmArgs = append(jvmArgs, "--compilation_level", "ADVANCED_OPTIMIZATIONS")
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

func ContextFor(w http.ResponseWriter, r *http.Request) Context {
  return w.(responseWriter).r
}

func NewRouter(shouldServe func(http.ResponseWriter, *http.Request) bool, notFound http.Handler) Router {
  if notFound == nil {
    notFound = http.NotFoundHandler()
  }

  if shouldServe == nil {
    shouldServe = func(w http.ResponseWriter, r *http.Request) bool {
      return true
    }
  }

  return &router{shouldServe, notFound, http.NewServeMux()}
}

type responseWriter struct {
  io.Writer
  http.ResponseWriter
  r *router
}

func (w responseWriter) Write(b []byte) (int, error) {
  return w.Writer.Write(b)
}

type router struct {
  shouldServe func(w http.ResponseWriter, r *http.Request) bool
  notFound http.Handler
  *http.ServeMux
}

func (d *router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  if !d.shouldServe(w, r) {
    return
  }

  // attempt to wrap ResponseWriter with gzip.
  if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
    g := gzip.NewWriter(w)
    defer g.Close()
    w.Header().Set("Content-Encoding", "gzip")
    w = responseWriter{g, w, d}
  } else if r.Header.Get("Connection") != "Upgrade" {
    w = responseWriter{w, w, d}
  }

  d.ServeMux.ServeHTTP(w, r)
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
  if strings.HasSuffix(filename, porkScriptFileExtension) {
    return porkscript
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
    source, found := findFile(d, changeTypeOfFile(path, javaScriptFileExtension, porkScriptFileExtension))
    if !found {
      c.ServeNotFound(w, r)
      return
    }
    w.Header().Set("Content-Type", "text/javascript")
    err := CompileJs(source, w, level)
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
      case porkscript:
        target, err := rebasePath(src, dst,
          changeTypeOfFile(path, porkScriptFileExtension, javaScriptFileExtension))
        if err != nil {
          return err
        }

        file, err := createFile(target)
        if err != nil {
          return err
        }
        defer file.Close()

        if err := CompileJs(path, file, h.level); err != nil {
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

func CompileJs(filename string, w io.Writer, level Optimization) error {
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

    jp, err = jsc(irp, owp, pathToJsc(), level)
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
