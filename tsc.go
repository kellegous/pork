package pork

import (
  "os"
  "os/exec"
)

func tscCommand(src, dst string) *exec.Cmd {
  c := exec.Command(PathToTsc, "--out", dst, src)
  c.Stderr = os.Stderr
  c.Stdout = os.Stdout
  return c
}

func CompileTsc(c *Config, src, dst string) error {
  return tscCommand(src, dst).Run()
}
