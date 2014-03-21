package pork

import (
  "io"
  "os"
)

func CompileJs(c *Config, src, dst string) error {
  r, err := os.Open(src)
  if err != nil {
    return err
  }
  defer r.Close()

  w, err := os.Create(dst)
  if err != nil {
    return err
  }
  defer w.Close()

  _, err = io.Copy(w, r)

  return err
}
