package main

import (
  "flag"
  "log"
  "net/http"
  "pork"
)

func main() {
  addr := flag.String("addr", ":6080", "The address to use")
  flag.Parse()

  var dirs []http.Dir
  if flag.NArg() == 0 {
    dirs = []http.Dir{http.Dir(".")}
  } else {
    dirs = make([]http.Dir, flag.NArg())
    for i, arg := range flag.Args() {
      dirs[i] = http.Dir(arg)
    }
  }

  // setup a simple router
  r := pork.NewRouter(func(status int, r *http.Request) {
    log.Printf("%d %s %s %s", status, r.RemoteAddr, r.Method, r.URL.String())
  }, nil, nil)
  r.Handle("/", pork.Content(pork.None, dirs...))
  http.ListenAndServe(*addr, r)
}