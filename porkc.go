package main

import (
  "fmt"
  "pork"
  "os"
)

// todo: actually make this do what it is supposed todo.
func main() {
  fmt.Println(os.Args[0])
  err := pork.Compile("tmp/foo.js", os.Stdout, "deps/jsc/compiler.jar", pork.Advanced)
  if err != nil {
    panic(err)
  }
}
