package pork

import (
  "io"
  "io/ioutil"
  "os"
)

func tscCommand(filename, tmpFile string, level Optimization) (*command, error) {
  return newCommand([]string{PathToTsc, "--out", tmpFile, filename}, "", nil)
}

func CompileTsc(c *Config, filename string, w io.Writer) error {
  t, err := ioutil.TempFile(os.TempDir(), "tsc-")
  if err != nil {
    return err
  }
  defer t.Close()
  defer os.Remove(t.Name())

  ca, err := tscCommand(filename, t.Name(), c.Level)
  if err != nil {
    return err
  }

  r, p, err := pipe(ca)
  if err != nil {
    return err
  }

  defer r.Close()

  if err := waitFor(p); err != nil {
    return err
  }

  switch c.Level {
  case None:
    if _, err := io.Copy(w, t); err != nil {
      return err
    }
  case Basic, Advanced:
    cb, err := jscCommand(nil, pathToJsc(), t.Name(), c.Level)
    if err != nil {
      return err
    }

    r, p, err := pipe(cb)
    if err != nil {
      return err
    }

    if _, err := io.Copy(w, r); err != nil {
      return err
    }

    if err := waitFor(p); err != nil {
      return err
    }
  }

  return nil
}
