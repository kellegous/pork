package pork

import (
  "bufio"
  "bytes"
  "fmt"
  "go/ast"
  "go/parser"
  "go/token"
  "io"
  "os"
  "path/filepath"
  "strconv"
  "strings"
)

// Execute an include directive
func execInclude(base, dir string, args []ast.Expr, w io.Writer) error {
  strs := make([]string, len(args))
  for i, arg := range args {

    bl, ok := arg.(*ast.BasicLit)
    if !ok || bl.Kind != token.STRING {
      return fmt.Errorf("expected string literal: %s", dir[arg.Pos()-1:arg.End()-1])
    }

    sv, err := strconv.Unquote(bl.Value)
    if err != nil {
      return err
    }

    strs[i] = sv
  }

  for _, str := range strs {
    if err := catFile(w, filepath.Join(base, str)); err != nil {
      return err
    }
  }

  return nil
}

// Expand an individual directive into the given writer.
func expandDirective(base, dir string, w io.Writer) error {
  e, err := parser.ParseExpr(dir)
  if err != nil {
    return err
  }

  c, ok := e.(*ast.CallExpr)
  if !ok {
    return fmt.Errorf("expected expression: %s", dir)
  }

  name := dir[c.Fun.Pos()-1 : c.Fun.End()-1]
  switch name {
  case "include":
    return execInclude(base, dir, c.Args, w)
  default:
    return fmt.Errorf("undefined directive: %s", name)
  }
}

// Expand all source directives into the given writer
func expandDirectives(filename string, w io.Writer) error {
  r, err := os.Open(filename)
  if err != nil {
    return err
  }
  defer r.Close()

  base := filepath.Dir(filename)

  br := bufio.NewReader(r)
  var buf bytes.Buffer
  for {
    b, p, err := br.ReadLine()
    if err == io.EOF {
      return nil
    } else if err != nil {
      return err
    }

    buf.Write(b)
    if p {
      continue
    }

    l := strings.TrimSpace(buf.String())
    buf.Reset()

    if len(l) > 0 && !strings.HasPrefix(l, "//") {
      return nil
    }

    if strings.HasPrefix(l, "//@") {
      if err := expandDirective(base, strings.TrimSpace(l[3:]), w); err != nil {
        return err
      }
    }
  }
}
