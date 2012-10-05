package pork

import (
  "bytes"
  "io"
  "path/filepath"
  "testing"
)

func levels() []Optimization {
  return []Optimization{None, Basic, Advanced}
}

func test(t *testing.T, fn func(*Config, string, io.Writer) error, c *Config, path string) {
  for _, level := range levels() {
    c.Level = level
    var b bytes.Buffer
    if err := fn(c, path, &b); err != nil {
      t.Error(err)
    }

    if len(b.String()) == 0 {
      t.Errorf("Empty output for level %d", level)
    }
  }
}

func TestJsx(t *testing.T) {
  test(t,
    CompileJsx,
    NewConfig(None),
    filepath.Join(Root(), "tests/jsx/a.jsx"))
}

func TestScss(t *testing.T) {
  test(t,
    CompileScss,
    NewConfig(None),
    filepath.Join(Root(), "tests/scss/a.scss"))
}
