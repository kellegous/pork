package main

import (
  "flag"
  "fmt"
  "github.com/kellegous/pork"
  "log"
  "net/http"
  "os"
)

func mainServe(args []string) {
  flags := flag.NewFlagSet("serve", flag.ExitOnError)
  flagAddr := flags.String("addr", ":8082", "address to bind")
  flags.Parse(args)

  var dirs []http.Dir
  if flags.NArg() == 0 {
    dirs = append(dirs, http.Dir("."))
  } else {
    for _, arg := range args {
      dirs = append(dirs, http.Dir(arg))
    }
  }

  r := pork.NewRouter(func(status int, r *http.Request) {
    log.Printf("[%d] %s", status, r.RequestURI)
  }, nil, nil)

  r.RespondWith("/", pork.Content(pork.NewConfig(pork.None), dirs...))

  if err := http.ListenAndServe(*flagAddr, r); err != nil {
    log.Panic(err)
  }
}

func mainBuild(args []string) {
  fmt.Println("TODO(knorton): Implement build.")
}

func main() {
  if len(os.Args) <= 1 {
    fmt.Println("help!")
    os.Exit(1)
  }

  switch os.Args[1] {
  case "serve":
    mainServe(os.Args[1:])
  case "build":
    mainBuild(os.Args[1:])
  default:
    fmt.Fprintln(os.Stderr, "Invalid command")
  }
}
