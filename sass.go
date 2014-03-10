package pork

import (
  "os"
  "os/exec"
  "path/filepath"
)

func pathToPluginsFile() string {
  return filepath.Join(rootDir, "sass_plugins.rb")
}

func sassCommand(c *Config, src, dst string) *exec.Cmd {
  args := []string{
    "--require", pathToPluginsFile(),
    "--no-cache",
    "--trace",
  }

  switch c.Level {
  case Basic, Advanced:
    args = append(args, "--style", "compressed")
  }

  for _, v := range c.ScssIncludes {
    args = append(args, "-I", v)
  }

  args = append(args, src, dst)
  cm := exec.Command(PathToSass, args...)
  cm.Stderr = os.Stderr
  cm.Stdout = os.Stdout
  return cm
}

func CompileScss(c *Config, src, dst string) error {
  return sassCommand(c, src, dst).Run()
}
