package pork

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Optimization ...
type Optimization int

const (
	// None ...
	None Optimization = iota

	// Basic ...
	Basic

	// Advanced ...
	Advanced
)

type srcType int

const (
	srcOfJsx srcType = iota
	srcOfTsc
	srcOfScss
	srcOfJs
	srcOfUnknown
)

type dstType int

const (
	dstOfJs dstType = iota
	dstOfCSS
	dstOfUnknown
)

const (
	// src
	jsxFileExtension  = ".main.jsx"
	tscFileExtension  = ".main.ts"
	scssFileExtension = ".main.scss"
	jsFileExtension   = ".main.js"

	// dst
	javaScriptFileExtension = ".js"
	cssFileExtension        = ".css"
)

var excludedSrcExtensions = []string{
	".ts",
	".jsx",
	".scss",
}

// PathToSass ...
var PathToSass = "sass"

// PathToJsx ...
var PathToJsx = "jsx"

// PathToTsc ...
var PathToTsc = "tsc"

// PathToClosureCompiler ...
var PathToClosureCompiler = "closure-compiler"

var rootDir string

func pathToJsc() string {
	return filepath.Join(rootDir, "deps/closure/compiler.jar")
}

// Router ...
type Router interface {
	RespondWith(string, Responder)
	RespondWithFunc(string, func(ResponseWriter, *http.Request))

	ServeHTTP(http.ResponseWriter, *http.Request)
}

// ResponseWriter ...
type ResponseWriter interface {
	http.ResponseWriter
	ServeNotFound()
	ServedFromPrefix() string
	EnableCompression()
}

type httpHandler struct {
	h http.Handler
}

func (h *httpHandler) ServePork(w ResponseWriter, r *http.Request) {
	h.h.ServeHTTP(w, r)
}

// ResponderFor ...
func ResponderFor(h http.Handler) Responder {
	return &httpHandler{h: h}
}

// NewRouter ...
func NewRouter(logger func(int, *http.Request), notFound Responder, headers map[string]string) Router {
	if notFound == nil {
		notFound = ResponderFor(http.NotFoundHandler())
	}

	if logger == nil {
		logger = func(status int, r *http.Request) {}
	}

	return &router{
		logger:   logger,
		notFound: notFound,
		headers:  headers,
		ServeMux: http.NewServeMux(),
	}
}

// The concrete implementation of pork's ResponseWriter
type response struct {
	http.ResponseWriter
	req    *http.Request
	writer io.Writer
	router *router
	status int
	prefix string
	closer io.Closer
}

func (r *response) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *response) Write(b []byte) (int, error) {
	return r.writer.Write(b)
}

func (r *response) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return r.ResponseWriter.(http.Hijacker).Hijack()
}

func (r *response) ServeNotFound() {
	if r.router != nil {
		r.router.notFound.ServePork(r, r.req)
	} else {
		http.NotFound(r, r.req)
	}
}

func (r *response) ServedFromPrefix() string {
	return r.prefix
}

func (r *response) EnableCompression() {
	// avoid double compressing
	if r.closer != nil {
		return
	}

	// avoid compressing if the client doesn't allow it
	if !strings.Contains(r.req.Header.Get("Accept-Encoding"), "gzip") {
		return
	}

	// insert an intermediate gzip writer
	g := gzip.NewWriter(r.writer)
	r.writer = g
	r.closer = g
	r.Header().Set("Content-Encoding", "gzip")
}

func (r *response) close() error {
	if r.closer == nil {
		return nil
	}
	return r.closer.Close()
}

type router struct {
	logger   func(status int, r *http.Request)
	notFound Responder
	headers  map[string]string
	*http.ServeMux
}

func (d *router) RespondWith(path string, r Responder) {
	d.ServeMux.Handle(path, &route{
		prefix:    path,
		responder: r,
		router:    d,
	})
}

func (d *router) RespondWithFunc(path string, f func(ResponseWriter, *http.Request)) {
	d.ServeMux.Handle(path, &route{
		prefix:    path,
		responder: ResponderFunc(f),
		router:    d,
	})
}

// Handler ...
type Handler interface {
	Responder
	Productionize(d http.Dir) (func() error, error)
}

// Responder ...
type Responder interface {
	ServePork(ResponseWriter, *http.Request)
}

type responderFunc func(ResponseWriter, *http.Request)

func (f responderFunc) ServePork(w ResponseWriter, r *http.Request) {
	f(w, r)
}

// ResponderFunc ...
func ResponderFunc(f func(ResponseWriter, *http.Request)) Responder {
	return responderFunc(f)
}

type route struct {
	prefix    string
	responder Responder
	router    *router
}

func (g *route) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// add any global headers
	h := w.Header()
	for k, v := range g.router.headers {
		h.Set(k, v)
	}

	// create a response object for the dispatch
	res := response{
		writer:         w,
		ResponseWriter: w,
		req:            r,
		router:         g.router,
		status:         200,
		prefix:         g.prefix,
	}

	// ensure that the response is flushed at the end
	defer res.close()

	// dispatch the request
	g.responder.ServePork(&res, r)

	// log the request
	g.router.logger(res.status, r)
}

type content struct {
	root []http.Dir
	conf *Config
	lock sync.RWMutex
}

func pathToThisFile() string {
	_, file, _, _ := runtime.Caller(0)
	return file
}

// Root ...
func Root() string {
	return rootDir
}

func init() {
	rootDir = filepath.Dir(pathToThisFile())
}

// HostRedirectResponder ...
type HostRedirectResponder string

// ServePork ...
func (h HostRedirectResponder) ServePork(w ResponseWriter, r *http.Request) {
	http.Redirect(w, r,
		fmt.Sprintf("http://%s%s", string(h), r.RequestURI),
		http.StatusMovedPermanently)
}

type errorFileResponder struct {
	path   string
	status int
}

func (c *errorFileResponder) ServePork(w ResponseWriter, r *http.Request) {
	w.EnableCompression()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(c.status)
	f, err := os.Open(c.path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	_, err = io.Copy(w, f)
	if err != nil {
		panic(err)
	}
}

// ErrorFileResponder ...
func ErrorFileResponder(path string, status int) Responder {
	if _, err := os.Stat(path); err != nil {
		panic(err)
	}
	return &errorFileResponder{
		path:   path,
		status: status,
	}
}

// ServeContentModificationTime returns true if content should be delivered
func ServeContentModificationTime(w http.ResponseWriter, r *http.Request, t time.Time) bool {
	if t.IsZero() {
		return true
	}

	// The Date-Modified header truncates sub-second precision, so
	// use mtime < t+1s instead of mtime <= t to check for unmodified.
	if ht, err := time.Parse(http.TimeFormat, r.Header.Get("If-Modified-Since")); err == nil && t.Before(ht.Add(1*time.Second)) {
		w.WriteHeader(http.StatusNotModified)
		return false
	}
	w.Header().Set("Last-Modified", t.UTC().Format(http.TimeFormat))
	return true
}

type fileResponder string

func (c fileResponder) ServePork(w ResponseWriter, r *http.Request) {
	w.EnableCompression()
	http.ServeFile(w, r, string(c))
}

// FileResponder ...
func FileResponder(path string) Responder {
	if _, err := os.Stat(path); err != nil {
		panic(err)
	}

	return fileResponder(path)
}

// Config ...
type Config struct {
	Level        Optimization
	JsxIncludes  []string
	JsxExterns   []string
	ScssIncludes []string
}

// NewConfig ...
func NewConfig(level Optimization) *Config {
	return &Config{
		Level: level,
	}
}

// Content ...
func Content(c *Config, d ...http.Dir) Handler {
	return &content{root: d, conf: c}
}

func expandPath(fs http.Dir, name string) string {
	return filepath.Join(string(fs), filepath.FromSlash(path.Clean("/"+name)))
}

type typeFound int

const (
	foundNothing typeFound = iota
	foundFile
	foundDirectory
)

func findFile(d []http.Dir, name string) (string, typeFound) {
	for i, n := 0, len(d); i < n; i++ {
		target := filepath.Join(string(d[i]), filepath.FromSlash(path.Clean("/"+name)))

		// if the file doesn't exist, move along
		s, err := os.Stat(target)
		if err != nil {
			continue
		}

		// if it's a file, return that
		if !s.IsDir() {
			return target, foundFile
		}

		// if it's a dir, check for an index
		target = filepath.Join(target, "index.html")
		if _, err := os.Stat(target); err == nil {
			return target, foundDirectory
		}
	}
	return "", foundNothing
}

func changeTypeOfFile(path, from, to string) string {
	// assert path ends with from
	return path[0:len(path)-len(from)] + to
}

func typeOfSrc(filename string) srcType {
	if strings.HasSuffix(filename, jsxFileExtension) {
		return srcOfJsx
	}
	if strings.HasSuffix(filename, tscFileExtension) {
		return srcOfTsc
	}
	if strings.HasSuffix(filename, jsFileExtension) {
		return srcOfJs
	}
	if strings.HasSuffix(filename, scssFileExtension) {
		return srcOfScss
	}
	return srcOfUnknown
}

func typeOfDst(filename string) dstType {
	switch filepath.Ext(filename) {
	case cssFileExtension:
		return dstOfCSS
	case javaScriptFileExtension:
		return dstOfJs
	}
	return dstOfUnknown
}

// Response ...
type Response struct {
	found   typeFound
	srcType srcType
	srcFile string
	req     *http.Request
}

// Deliver ...
func (r *Response) Deliver(cfg *Config, w ResponseWriter) {
	path := r.req.URL.Path
	switch r.srcType {
	case srcOfUnknown:
		if r.found == foundDirectory && path[len(path)-1] != '/' {
			http.Redirect(w, r.req, path+"/", http.StatusMovedPermanently)
			return
		}
		w.EnableCompression()
		http.ServeFile(w, r.req, r.srcFile)
	case srcOfJsx:
		w.EnableCompression()
		w.Header().Set("Content-Type", "text/javascript")
		if err := compile(cfg, r.srcFile, w, CompileJsx, optimizeJs); err != nil {
			panic(err)
		}
	case srcOfTsc:
		w.EnableCompression()
		w.Header().Set("Content-Type", "text/javascript")
		if err := compile(cfg, r.srcFile, w, CompileTsc, optimizeJs); err != nil {
			panic(err)
		}
	case srcOfJs:
		w.EnableCompression()
		w.Header().Set("Content-Type", "text/javascript")
		if err := compile(cfg, r.srcFile, w, CompileJs, optimizeJs); err != nil {
			panic(err)
		}
	case srcOfScss:
		w.EnableCompression()
		w.Header().Set("Content-Type", "text/css")
		if err := compile(cfg, r.srcFile, w, CompileScss, optimizeCss); err != nil {
			panic(err)
		}
	default:
		panic("unknown src type")
	}
}

// FindContent ...
func FindContent(prefix string, r *http.Request, d ...http.Dir) (*Response, error) {
	pth := r.URL.Path
	rel, err := filepath.Rel(prefix, pth)
	if err != nil {
		return nil, err
	}

	// if the file exists, create a direct response
	if target, found := findFile(d, rel); found != foundNothing {
		return &Response{
			found:   found,
			srcType: srcOfUnknown,
			srcFile: target,
			req:     r,
		}, nil
	}

	switch typeOfDst(rel) {
	case dstOfJs:
		// try to answer with jsx
		jsxSrc, found := findFile(d, changeTypeOfFile(rel, javaScriptFileExtension, jsxFileExtension))
		if found == foundFile {
			return &Response{
				found:   found,
				srcType: srcOfJsx,
				srcFile: jsxSrc,
				req:     r,
			}, nil
		}

		tscSrc, found := findFile(d, changeTypeOfFile(rel, javaScriptFileExtension, tscFileExtension))
		if found == foundFile {
			return &Response{
				found:   found,
				srcType: srcOfTsc,
				srcFile: tscSrc,
				req:     r,
			}, nil
		}

		jsSrc, found := findFile(d, changeTypeOfFile(rel, javaScriptFileExtension, jsFileExtension))
		if found == foundFile {
			return &Response{
				found:   found,
				srcType: srcOfJs,
				srcFile: jsSrc,
				req:     r,
			}, nil
		}

	case dstOfCSS:
		cssSrc, found := findFile(d, changeTypeOfFile(rel, cssFileExtension, scssFileExtension))
		if found == foundFile {
			return &Response{
				found:   found,
				srcType: srcOfScss,
				srcFile: cssSrc,
				req:     r,
			}, nil
		}
	}
	return nil, nil
}

// ServeContent ...
func ServeContent(w ResponseWriter, r *http.Request, cfg *Config, d ...http.Dir) {
	res, err := FindContent(w.ServedFromPrefix(), r, d...)
	if err != nil {
		panic(err)
	}

	if res != nil {
		res.Deliver(cfg, w)
		return
	}

	w.ServeNotFound()
}

func (h *content) ServePork(w ResponseWriter, r *http.Request) {
	h.lock.RLock()
	defer h.lock.RUnlock()
	ServeContent(w, r, h.conf, h.root...)
}

func rebasePath(src, dst, filename string) (string, error) {
	target, err := filepath.Rel(src, filename)
	if err != nil {
		return "", err
	}
	return filepath.Join(dst, target), nil
}

func catFile(w io.Writer, filename string) error {
	r, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer r.Close()

	if _, err := io.Copy(w, r); err != nil {
		return err
	}

	return nil
}

func compile(c *Config, src string, w io.Writer,
	cmp func(*Config, string, string) error,
	opt func(*Config, io.Writer) (io.WriteCloser, error)) error {

	// create an optimization pipe
	wo, err := opt(c, w)
	if err != nil {
		return err
	}
	defer wo.Close()

	// open a temp file for the base compilation
	t, err := ioutil.TempFile(os.TempDir(), "cmp-")
	if err != nil {
		return err
	}
	defer t.Close()
	defer os.Remove(t.Name())

	// TODO(knorton): This can be executed in parallel with
	// directive expansion. It just needs to return the underlying
	// os.Process which allows for Wait.
	if err := cmp(c, src, t.Name()); err != nil {
		return err
	}

	// expand source directives
	if err := expandDirectives(src, wo); err != nil {
		return err
	}

	// copy the compile output into the writer
	return catFile(wo, t.Name())
}

func ensureDir(dir string) error {
	if _, err := os.Stat(dir); err != nil {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return err
		}
	}
	return nil
}

func compileToFile(c *Config, src, dst string,
	cmp func(*Config, string, string) error,
	opt func(*Config, io.Writer) (io.WriteCloser, error)) error {
	dir := filepath.Dir(dst)

	if err := ensureDir(dir); err != nil {
		return err
	}

	file, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer file.Close()

	return compile(c, src, file, cmp, opt)
}

func copyFile(dst, src string) error {
	if err := ensureDir(filepath.Dir(dst)); err != nil {
		return err
	}

	r, err := os.Open(src)
	if err != nil {
		return err
	}
	defer r.Close()

	w, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer w.Close()

	if _, err := io.Copy(w, r); err != nil {
		return err
	}

	return nil
}

func isExcludedSrc(path string) bool {
	for _, suf := range excludedSrcExtensions {
		if strings.HasSuffix(path, suf) {
			return true
		}
	}
	return false
}

func productionize(cfg *Config, roots []http.Dir, dest http.Dir) error {
	d := string(dest)
	if _, err := os.Stat(d); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err := os.MkdirAll(d, os.ModePerm); err != nil {
			return err
		}
	}

	for _, root := range roots {
		src := string(root)
		if err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
			switch typeOfSrc(path) {
			case srcOfJsx:
				target, err := rebasePath(src, d,
					changeTypeOfFile(path, jsxFileExtension, javaScriptFileExtension))
				if err != nil {
					return err
				}

				if err := compileToFile(cfg, path, target, CompileJsx, optimizeJs); err != nil {
					return err
				}
			case srcOfTsc:
				target, err := rebasePath(src, d,
					changeTypeOfFile(path, tscFileExtension, javaScriptFileExtension))
				if err != nil {
					return err
				}

				if err := compileToFile(cfg, path, target, CompileTsc, optimizeJs); err != nil {
					return err
				}
			case srcOfJs:
				target, err := rebasePath(src, d,
					changeTypeOfFile(path, jsFileExtension, javaScriptFileExtension))
				if err != nil {
					return err
				}

				if err := compileToFile(cfg, path, target, CompileJs, optimizeJs); err != nil {
					return err
				}
			case srcOfScss:
				target, err := rebasePath(src, d,
					changeTypeOfFile(path, scssFileExtension, cssFileExtension))
				if err != nil {
					return err
				}

				if err := compileToFile(cfg, path, target, CompileScss, optimizeCss); err != nil {
					return err
				}
			default:
				if !info.IsDir() && !isExcludedSrc(path) {
					target, err := rebasePath(src, d, path)
					if err != nil {
						return err
					}

					if err := copyFile(target, path); err != nil {
						return err
					}
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}

func (h *content) Productionize(d http.Dir) (func() error, error) {
	h.lock.Lock()
	defer h.lock.Unlock()

	if err := productionize(h.conf, h.root, d); err != nil {
		return nil, err
	}

	// prepend the dest dir to the roots
	root := make([]http.Dir, len(h.root)+1)
	root[0] = d
	copy(root[1:], h.root)
	h.root = root

	return func() error {
		h.lock.Lock()
		defer h.lock.Unlock()
		return productionize(h.conf, h.root, d)
	}, nil
}
