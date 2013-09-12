package pork

import (
  "io"
)

func sassCommand(filename string, level Optimization) (*command, error) {
  args := []string{
    PathToSass,
    "--no-cache",
    "--trace"}

  switch level {
  case Basic:
    args = append(args, "--style", "compact")
  case Advanced:
    args = append(args, "--style", "compressed")
  }

  args = append(args, filename)
  return newCommand(args, "", nil)
}

// todo: add cpp to the frontend of this
func CompileScss(c *Config, filename string, w io.Writer) error {
  cs, err := sassCommand(filename, c.Level)
  if err != nil {
    return err
  }

  r, p, err := pipe(cs)
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
