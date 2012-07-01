package pork

import (
  "io"
  "path/filepath"
)

func pathToSass() string {
  return filepath.Join(rootDir, "sass")
}

func sassCommand(filename string, sassPath string, level Optimization) *command {
  args := []string{
    PathToRuby,
    sassPath,
    "--no-cache",
    "--trace"}

  switch level {
  case Basic:
    args = append(args, "--style", "compact")
  case Advanced:
    args = append(args, "--style", "compressed")
  }

  args = append(args, filename)
  return &command{args, "", nil}
}

// todo: add cpp to the frontend of this
func CompileScss(c *Config, filename string, w io.Writer) error {
  r, p, err := pipe(sassCommand(filename, pathToSass(), c.Level))
  if err != nil {
    return err
  }
  defer r.Close()

  _, err = io.Copy(w, r)
  if err != nil {
    return err
  }

  if err := waitFor(p); err != nil {
    return err
  }

  return nil
}
