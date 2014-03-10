package pork

import (
  "os"
  "os/exec"
  "path/filepath"
)

func jsxCommand(c *Config, src, dst string) *exec.Cmd {
  args := []string{"--output", dst}

  for _, i := range c.JsxIncludes {
    args = append(args, "--add-search-path", i)
  }

  switch c.Level {
  case Basic:
    args = append(args, "--release")
  case Advanced:
    args = append(args,
      "--release",
      "--optimize",
      "no-assert,no-log,inline,return-if")
  }

  args = append(args, filepath.Base(src))
  cm := exec.Command(PathToJsx, args...)

  // For jsx, we execute with a difference cwd to avoid having
  // absolute paths in the class map.
  cm.Dir = filepath.Dir(src)

  cm.Stderr = os.Stderr
  cm.Stdout = os.Stdout
  return cm
}

func CompileJsx(c *Config, src, dst string) error {
  return jsxCommand(c, src, dst).Run()
}
