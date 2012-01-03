package pork

import (
  "errors"
  "fmt"
  "net/http"
  "io"
  "os"
  "path"
  "path/filepath"
  "strings"
)

// todo: implement Content.Productionize
// todo: implement multi-mount.
type Optimization int
const (
  None Optimization = iota
  Basic
  Advanced
)

type fileType int
const (
  javascript fileType = iota
  css
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

func waitFor(p *os.Process) error {
  s, err := p.Wait(0)
  if err != nil {
    return err
  }

  if sc := s.WaitStatus.ExitStatus(); sc != 0 {
    return errors.New(fmt.Sprintf("exit code: %d", sc))
  }

  return nil
}

func cpp(filename string, w *os.File) (*os.Process, error) {
  cppArgs := []string{
    PathToCpp,
    "-P",
    "-CC",
    fmt.Sprintf("-I%s", filepath.Join(rootDir, "js")),
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

type content struct {
  root []http.Dir
  level Optimization
}

func Init(root string) {
  r, err := filepath.Abs(root)
  if err != nil {
    panic(err)
  }
  rootDir = r
}

func Content(level Optimization, d ...http.Dir) http.Handler {
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

func typeOfFile(filename string) fileType {
  if strings.HasSuffix(filename, javaScriptFileExtension) {
    return javascript
  }
  if strings.HasSuffix(filename, cssFileExtension) {
    return css
  }
  return unknown
}

func ServeContent(w http.ResponseWriter, r *http.Request, level Optimization, d ...http.Dir) {
  path := r.URL.Path

  // if the file exists, just serve it.
  if target, found := findFile(d, path); found {
    http.ServeFile(w, r, target)
    return
  }

  switch typeOfFile(path) {
  case javascript:
    source, found := findFile(d, path[0 : len(path) - len(javaScriptFileExtension)] + porkScriptFileExtension)
    if !found {
      ServeNotFound(w, r)
      return
    }
    w.Header().Set("Content-Type", "text/javascript")
    err := CompileJs(source, w, level)
    if err != nil {
      // todo: send to ServeSiteError
      panic(err)
    }
  case css:
    source, found := findFile(d, path[0 : len(path) - len(cssFileExtension)] + sassFileExtension)
    if !found {
      ServeNotFound(w, r)
      return
    }
    w.Header().Set("Content-Type", "text/css")
    err := CompileCss(source, w)
    if err != nil {
      // todo: send to ServeSiteError
      panic(err)
    }
  default:
    ServeNotFound(w, r)  
  }
}

func (h *content) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  ServeContent(w, r, h.level, h.root...)
}

func ServeNotFound(w http.ResponseWriter, r *http.Request) {
  http.NotFound(w, r)
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

  var cp *os.Process
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

    jp, err := jsc(irp, owp, pathToJsc(), level)
    if err != nil {
      return err
    }

    irp.Close()
    owp.Close()

    err = waitFor(jp)
    if err != nil {
      return err
    }
  }

  err = waitFor(cp)
  if err != nil {
    return err
  }

  _, err = io.Copy(w, orp)
  if err != nil {
    return err
  }

  return nil
}
