package pork

import (
  "fmt"
  "io"
  "os"
  "path/filepath"
)

func jsxCommand(filename string, includes []string, level Optimization) (*command, error) {
  args := []string{PathToJsx}
  for _, i := range includes {
    args = append(args, "--add-search-path", i)
  }
  switch level {
  case Basic:
    args = append(args, "--release")
  case Advanced:
    args = append(args,
      "--release",
      "--optimize",
      "no-assert,no-log,inline,return-if")
  }
  // For jsx, we execute with a difference cwd to avoid having
  // absolute paths in the class map.
  args = append(args, filepath.Base(filename))
  return newCommand(args, filepath.Dir(filename), nil)
}

// todo: add cpp to the front-end of this.
func CompileJsx(c *Config, filename string, w io.Writer) error {
  var p []*os.Process
  var r io.ReadCloser
  var err error

  ca, err := jsxCommand(filename, c.JsxIncludes, c.Level)
  if err != nil {
    return err
  }

  switch c.Level {
  case None:
    r, p, err = pipe(ca)
    if err != nil {
      return err
    }
    defer r.Close()
  case Basic, Advanced:
    cb, err := jscCommand(c.JsxExterns, pathToJsc(), "", c.Level)
    if err != nil {
      return err
    }

    r, p, err = pipe(ca, cb)
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

  // this is a hack for now.
  _, err = fmt.Fprintf(w,
    "JSX.require(\"%s\")._Main.main$([]);\n",
    filepath.Base(filename))
  if err != nil {
    return err
  }

  return nil
}
