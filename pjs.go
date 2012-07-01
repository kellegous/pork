package pork

import (
  "io"
  "os"
)

// Compiles a pork-enhanced JavaScript target, writing the results
// into w.
func CompilePjs(c *Config, filename string, w io.Writer) error {
  var p []*os.Process
  var r io.ReadCloser
  var err error

  switch c.Level {
  case None:
    r, p, err = pipe(cppCommand(filename, c.PjsIncludes))
    if err != nil {
      return err
    }
    defer r.Close()
  case Basic, Advanced:
    r, p, err = pipe(
      cppCommand(filename, c.PjsIncludes),
      jscCommand(c.PjsExterns, pathToJsc(), c.Level))
    if err != nil {
      return err
    }
    defer r.Close()
  }

  _, err = io.Copy(w, r)
  if err != nil {
    return err
  }

  if err := waitFor(p); err != nil {
    return err
  }

  return nil
}
