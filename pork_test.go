package pork

import (
  "bytes"
  "errors"
  "io"
  "path/filepath"
  "strings"
  "testing"
)

func levels() []Optimization {
  return []Optimization{None, Basic, Advanced}
}

func test(t *testing.T,
  fn func(*Config, string, io.Writer) error,
  c *Config, path string,
  check func([]byte) error) {
  for _, level := range levels() {
    c.Level = level
    var b bytes.Buffer
    if err := fn(c, path, &b); err != nil {
      t.Error(err)
    }

    if len(b.String()) == 0 {
      t.Errorf("Empty output for level %d", level)
    }

    if check == nil {
      continue
    }

    if err := check(b.Bytes()); err != nil {
      t.Error(err)
    }
  }
}

func TestJsx(t *testing.T) {
  test(t,
    CompileJsx,
    NewConfig(None),
    filepath.Join(Root(), "tests/jsx/a.jsx"),
    nil)
}

func TestScss(t *testing.T) {
  test(t,
    CompileScss,
    NewConfig(None),
    filepath.Join(Root(), "tests/scss/a.scss"),
    nil)

  test(t,
    CompileScss,
    NewConfig(None),
    filepath.Join(Root(), "tests/scss/datauri.scss"),
    func(b []byte) error {
      if !strings.Contains(string(b), "data:image/png;base64") {
        return errors.New("datauri did not produce base64")
      }
      return nil
    })
}

func TestTsc(t *testing.T) {
  test(t,
    CompileTsc,
    NewConfig(None),
    filepath.Join(Root(), "tests/tsc/a.ts"),
    nil)
}
