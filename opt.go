package pork

import (
  "io"
  "os/exec"
)

type jsOpt struct {
  io.Writer
  cm *exec.Cmd
  wc io.Closer
}

func (o *jsOpt) Close() error {
  if err := o.wc.Close(); err != nil {
    return err
  }

  return o.cm.Wait()
}

type noOpt struct {
  io.Writer
}

func (o *noOpt) Close() error {
  return nil
}

func jscCommand(externs []string, jscPath string, level Optimization) *exec.Cmd {
  args := []string{"-jar", jscPath, "--language_in", "ECMASCRIPT5"}

  switch level {
  case Basic:
    args = append(args, "--compilation_level", "SIMPLE_OPTIMIZATIONS")
  case Advanced:
    args = append(args, "--compilation_level", "ADVANCED_OPTIMIZATIONS")
  }

  for _, e := range externs {
    args = append(args, "--externs", e)
  }

  return exec.Command(PathToJava, args...)
}

func optimizeJs(c *Config, w io.Writer) (io.WriteCloser, error) {
  switch c.Level {
  case Basic, Advanced:
    cm := jscCommand(c.JsxExterns, pathToJsc(), c.Level)

    wc, err := cm.StdinPipe()
    if err != nil {
      return nil, err
    }

    return &jsOpt{
      Writer: w,
      cm:     cm,
      wc:     wc,
    }, nil
  }
  return &noOpt{Writer: w}, nil
}

func optimizeCss(c *Config, w io.Writer) (io.WriteCloser, error) {
  return &noOpt{Writer: w}, nil
}
