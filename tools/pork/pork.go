package main

import (
  "flag"
  "github.com/kellegous/pork"
  "log"
  "net/http"
)

func main() {
  flagAddr := flag.String("addr", ":8082", "address to bind")

  flag.Parse()

  var dirs []http.Dir
  if flag.NArg() == 0 {
    dirs = append(dirs, http.Dir("."))
  } else {
    for _, arg := range flag.Args() {
      dirs = append(dirs, http.Dir(arg))
    }
  }

  r := pork.NewRouter(func(status int, r *http.Request) {
    log.Printf("[%d] %s", status, r.RequestURI)
  }, nil, nil)
  r.Handle("/", pork.Content(pork.NewConfig(pork.None), dirs...))

  if err := http.ListenAndServe(*flagAddr, r); err != nil {
    log.Panic(err)
  }
}
