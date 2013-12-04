package main

import (
  "errors"
  "flag"
  "fmt"
  "github.com/kellegous/pork"
  "log"
)

func main() {
  tomatoIn := flag.String("tomatoIn", "views", "the folder to use as the tomato input root folder")
  tomatoOut := flag.String("tomatoOut", "gen/views.ts", "the output file to emit generated tomato views to")
  language := flag.String("language", "ts", "what language to use for the generated tomato views")
  qImport := flag.String("qImport", "../ts/util/q.ts", "where to find the library for q")
  forceDebugIds := flag.Bool("debugIds", false, "whether or not to force generated Views to have debug-ids")

  flag.Parse()

  if err := pork.GenerateTomatoes(*tomatoIn, *tomatoOut, getLanguage(*language), *qImport, *forceDebugIds); err != nil {
    fmt.Println(err.Error())
  }
}

func getLanguage(language string) pork.Language {
  // TODO(jaime): support other languages
  if "ts" != language {
    log.Panic(errors.New("That language is currently not supported!"))
  }

  return pork.TypeScript
}
