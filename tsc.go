package pork

import (
  "io"
  "io/ioutil"
  "path/filepath"
  "os"
)

func pathToTsc() string {
  return filepath.Join(rootDir, "deps/tsc/bin/tsc")
}

func tscCommand(filename, tscPath, tmpFile string, level Optimization) *command {
  return &command{
    []string{
      PathToNode,
      tscPath,
      "--out", tmpFile,
      filename,
    },
    "",
    nil,
  }
}

func CompileTsc(c *Config, filename string, w io.Writer) error {
  // TODO(knorton): For now only, let's just do basic.
  t, err := ioutil.TempFile(os.TempDir(), "tsc-")
  if err != nil {
    return err
  }
  defer t.Close()
  defer os.Remove(t.Name())

  r, p, err := pipe(tscCommand(filename, pathToTsc(), t.Name(), c.Level))
  if err != nil {
    return err
  }

  defer r.Close()

  if err := waitFor(p); err != nil {
    return err
  }

  if _, err := io.Copy(w, t); err != nil {
    return err
  }

  return nil
}
