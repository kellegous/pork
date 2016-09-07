package pork

import (
  "io"
  "os"
  "os/exec"
)

type jsOpt struct {
  io.WriteCloser
  cm *exec.Cmd
}

func (o *jsOpt) Close() error {
  if err := o.WriteCloser.Close(); err != nil {
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
  args := []string{"--language_in", "ECMASCRIPT5"}

  switch level {
  case Basic:
    args = append(args, "--compilation_level", "SIMPLE_OPTIMIZATIONS")
  case Advanced:
    args = append(args, "--compilation_level", "ADVANCED_OPTIMIZATIONS")
  }

  for _, e := range externs {
    args = append(args, "--externs", e)
  }

  return exec.Command(PathToClosureCompiler, args...)
}

// Creates an optimization pipe for JavaScript streams
func optimizeJs(c *Config, w io.Writer) (io.WriteCloser, error) {
  switch c.Level {
  case Basic, Advanced:
    cm := jscCommand(c.JsxExterns, pathToJsc(), c.Level)

    // connect the output of the command to the writer
    cm.Stdout = w

    // send all error spew to the orginal stderr stream
    cm.Stderr = os.Stderr

    // open a pipe into stdin
    wc, err := cm.StdinPipe()
    if err != nil {
      return nil, err
    }

    // fire that off in the background
    if err := cm.Start(); err != nil {
      return nil, err
    }

    return &jsOpt{
      WriteCloser: wc,
      cm:          cm,
    }, nil
  }
  return &noOpt{Writer: w}, nil
}

// Creates an optimization pipe for JavaScript streams
func optimizeCss(c *Config, w io.Writer) (io.WriteCloser, error) {
  return &noOpt{Writer: w}, nil
}
