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

  // Run tomato if we have set a configuration for it.
  if c.TomatoConfig != nil {
    tConf := c.TomatoConfig
    if err := GenerateTomatoes(tConf.TomatoRoot, tConf.TomatoDst, TypeScript, tConf.QImport, tConf.ForceDebugIds); err != nil {
      return err
    }
  }

  // TODO(knorton): For now only, let's just do basic.
  t, err := ioutil.TempFile(os.TempDir(), "tsc-")
  if err != nil {
    return err
  }
  defer t.Close()
  defer os.Remove(t.Name())

  cmd, err := tscCommand(filename, t.Name(), c.Level)
  if err != nil {
    return err
  }

  r, p, err := pipe(cmd)
  if err != nil {
    return err
  }

  defer r.Close()

  if err := waitFor(p); err != nil {
    return err
  }

  if _, err := io.Copy(w, t); err != nil {
    return err
  }

  return nil
}
