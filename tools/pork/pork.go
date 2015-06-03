package main

import (
	"flag"
	"fmt"
	"github.com/kellegous/pork"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

func print(w io.Writer, lines []string) {
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
}

func parseOptimization(v string) (pork.Optimization, error) {
	switch strings.ToLower(v) {
	case "none":
		return pork.None, nil
	case "basic":
		return pork.Basic, nil
	case "advanced":
		return pork.Advanced, nil
	}
	return pork.None, fmt.Errorf("invalid optimization level: %s", v)
}

func helpServe(w io.Writer) {
	print(w, []string{
		"  pork serve [options] dir...",
		"",
		"  options:",
		"  --addr=addr    the address to which the http server will bind (default: \":8082\")",
		"  --opt=level    the pork optimization level (None, Basic, Advanced)",
		"",
	})
}

func mainServe(args []string) {
	flags := flag.NewFlagSet("", flag.ExitOnError)
	flagAddr := flags.String("addr", ":8082", "address to bind")
	flags.Parse(args)

	var dirs []http.Dir
	if flags.NArg() == 0 {
		dirs = append(dirs, http.Dir("."))
	} else {
		for _, arg := range flags.Args() {
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

func helpBuild(w io.Writer) {
	print(w, []string{
		"  pork build [options] dir...",
		"",
		"  options:",
		"  --out=path     the path to write the output. the default is to write into the pork directory.",
		"  --opt=level    the pork optimization level (None, Basic, Advanced)",
		"",
	})
}

func mainBuild(args []string) {
	flags := flag.NewFlagSet("", flag.ExitOnError)
	flagOut := flags.String("out", "", "")
	flagOpt := flags.String("opt", "None", "")
	flags.Parse(args)

	lvl, err := parseOptimization(*flagOpt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid optimization level: %s\n", *flagOpt)
	}

	var dirs []http.Dir
	if flags.NArg() == 0 {
		dirs = append(dirs, http.Dir("."))
	} else {
		for _, arg := range flags.Args() {
			dirs = append(dirs, http.Dir(arg))
		}
	}

	for _, dir := range dirs {
		out := dir
		if *flagOut != "" {
			out = http.Dir(*flagOut)
		}

		if _, err := pork.Content(pork.NewConfig(lvl), dir).Productionize(out); err != nil {
			log.Panic(err)
		}
	}
}

func helpMain(w io.Writer) {
	print(w, []string{
		"  pork command [options] args...",
		"",
		"  commands:",
		"  serve        run a porkifying http server on one or more pork directories",
		"  build        productionize one or more pork directories",
		"  help         get help on one of these here commands",
		"",
	})
}

func printHelp(w io.Writer, topic string) {
	switch strings.ToLower(topic) {
	case "serve":
		helpServe(w)
	case "build":
		helpBuild(w)
	default:
		helpMain(w)
	}
	os.Exit(1)
}

func main() {
	if len(os.Args) <= 1 {
		printHelp(os.Stderr, "")
	}

	switch os.Args[1] {
	case "serve":
		mainServe(os.Args[2:])
	case "build":
		mainBuild(os.Args[2:])
	case "help":
		var t string
		if len(os.Args) >= 3 {
			t = os.Args[2]
		}
		printHelp(os.Stderr, t)
	default:
		print(os.Stderr, []string{
			fmt.Sprintf("invalid command: %s", os.Args[1]),
			"",
		})
		printHelp(os.Stderr, "")
	}
}
