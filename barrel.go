package pork

import (
  "encoding/json"
  "errors"
  "fmt"
  "io"
  "os"
  "path/filepath"
)

type barrel struct {
  Units []*unit
}

type unit struct {
  File     string
  LevelStr string `json:"Level"`
  Externs  []string
  Includes []string
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

func minLevel(a Optimization, b Optimization) Optimization {
  if a < b {
    return a
  }
  return b
}

func appendPaths(base string, into, from []string) []string {
  for _, p := range from {
    into = append(into, filepath.Join(base, p))
  }
  return into
}

func CompileBarrel(c *Config, filename string, w io.Writer) error {
  dir := filepath.Dir(filename)

  file, err := os.Open(filename)
  if err != nil {
    return err
  }
  defer file.Close()

  barrel := &barrel{}
  if err := json.NewDecoder(file).Decode(&barrel); err != nil {
    return err
  }

  for _, unit := range barrel.Units {
    n := *c
    n.Level = minLevel(unit.Level(), c.Level)
    switch typeOfSrc(unit.File) {
    case srcOfJsx:
      n.JsxIncludes = appendPaths(dir, n.JsxIncludes, unit.Includes)
      n.JsxExterns = appendPaths(dir, n.JsxExterns, unit.Externs)
      if err := CompileJsx(&n, filepath.Join(dir, unit.File), w); err != nil {
        return err
      }
    case srcOfPjs:
      n.PjsIncludes = appendPaths(dir, n.PjsIncludes, unit.Includes)
      n.PjsExterns = appendPaths(dir, n.PjsExterns, unit.Externs)
      if err := CompilePjs(&n, filepath.Join(dir, unit.File), w); err != nil {
        return err
      }
    default:
      return errors.New(fmt.Sprintf("Uknown file type for %s", filename))
    }
  }
  return nil
}
