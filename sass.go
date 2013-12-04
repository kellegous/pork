package pork

import (
  "io"
  "path/filepath"
)

func pathToPluginsFile() string {
  return filepath.Join(rootDir, "sass_plugins.rb")
}

func sassCommand(c *Config, filename string) (*command, error) {
  args := []string{
    PathToSass,
    "--require", pathToPluginsFile(),
    "--no-cache",
    "--trace"}

  switch c.Level {
  case Basic:
    args = append(args, "--style", "compact")
  case Advanced:
    args = append(args, "--style", "compressed")
  }

  for _, v := range c.ScssIncludes {
    args = append(args, "-I", v)
  }

  args = append(args, filename)
  return newCommand(args, "", nil)
}

// todo: add cpp to the frontend of this
func CompileScss(c *Config, filename string, w io.Writer) error {
  cs, err := sassCommand(c, filename)
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
