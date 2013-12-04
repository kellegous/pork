package pork

import (
  "bufio"
  "bytes"
  "code.google.com/p/go.net/html"
  "container/list"
  "errors"
  "fmt"
  "os"
  "strings"
)

type Language int

const (
  TypeScript Language = iota
)

// Special attributes on tomato template elements
const (
  FieldRefAttr    = "_ref"
  MockAttr        = "_ignorecontent"
  TunnelledIdAttr = "_id"
  IdAttr          = "id"
  DebugIdAttr     = "debug-id"
)

// TODO(jaime): Wish I could make this const
// List of attributes we do not forward into the generated JSX.
var blockedAttrs = []string{FieldRefAttr, MockAttr, IdAttr}

type TomatoGenerator interface {
  GenerateViews(files *list.List, forceDebugIds bool) (map[string]string, error)
  EmitPreamble(buffer *bytes.Buffer)
  EmitPostamble(buffer *bytes.Buffer)
  generateView(fileName string, forceDebugIds bool) (string, error)
}

type generatorData struct {
  root    string
  qImport string
}

type viewGenerator interface {
  // Visitor to build up the string
  head(node *html.Node, depth int) error
  tail(node *html.Node, depth int)
  transferAttrs(node *html.Node)

  // View emitting.
  emitPreamble()
  emitElementRefs()
  emitDomConstruction()
  emitPostamble()
  getView() string
}

type visitorData struct {
  viewName        string
  output          stringBuilder
  domConstruction stringBuilder
  ignoreSubtree   bool
  forceDebugIds   bool
  refs            list.List
  appendStack     list.List
}

// Factory method for obtaining a TomatoGenerator
func MakeTomatoGenerator(root string, language Language, qImport string) (TomatoGenerator, error) {
  // TODO(jaime): Support the other languages that pork supports. Someday over the rainbow.
  switch language {
  case TypeScript:
    return &typeScriptGenerator{generatorData{root, qImport}}, nil
  default:
    return nil, errors.New("Language not supported")
  }
}

// Utility for building up Strings in memory efficiently.
type stringBuilder struct {
  buffer bytes.Buffer
}

func (builder *stringBuilder) append(text string) *stringBuilder {
  builder.buffer.WriteString(text)
  return builder
}

///////////////////
// TYPESCRIPT IMPL
//////////////////

type typeScriptVisitor struct {
  visitorData // inherits
}

type typeScriptGenerator struct {
  generatorData //inherits
}

func (g *typeScriptGenerator) EmitPreamble(buffer *bytes.Buffer) {
  buffer.WriteString("/// <reference path=\"")
  buffer.WriteString(g.qImport)
  // TODO(jaime): Make this module parameterizable. Views depend on the q library, so for now
  // it makes sense to shove them into the same module.
  buffer.WriteString("\" />\n\nmodule Q {\n\n")
}

func (g *typeScriptGenerator) GenerateViews(files *list.List, forceDebugIds bool) (map[string]string, error) {
  views := make(map[string]string)
  for e := files.Front(); e != nil; e = e.Next() {
    file := e.Value.(string)
    view, err := g.generateView(file, forceDebugIds)
    if err != nil {
      return nil, err
    }
    views[file] = view
  }
  return views, nil
}

func (*typeScriptGenerator) EmitPostamble(buffer *bytes.Buffer) {
  buffer.WriteString("} // module Q\n")
}

func (*typeScriptGenerator) generateView(fileName string, forceDebugIds bool) (string, error) {
  visitor := typeScriptVisitor{visitorData{
    forceDebugIds: forceDebugIds,
    viewName:      getViewName(fileName),
  }}

  if err := walk(fileName, &visitor); err != nil {
    return "", err
  }

  // Generate the View and return it.
  return generateView(&visitor), nil
}

// DF going down the stack.
func (v *typeScriptVisitor) head(node *html.Node, depth int) error {
  if v.ignoreSubtree {
    return nil
  }

  switch node.Type {
  case html.ElementNode:
    tagName := strings.ToLower(node.Data)
    v.domConstruction.append(indent(depth))

    if depth == 0 {

      // This is the first part of the view (call to super constructor).
      v.domConstruction.append("super(document.createElement('").append(tagName).append("'));\n").append(indent(depth)).append("this")

      // Include debug IDs if we force them to.
      if v.forceDebugIds && !hasAttr(node, DebugIdAttr) {
        emitAttr(&v.domConstruction, DebugIdAttr, debugIdFromViewName(v.viewName))
      }
    } else {

      // A sub-element. Lets start a call to append.
      v.appendStack.PushBack(node)
      v.domConstruction.append(".append(")

      // Is this element one that we need to elevate to a field reference?
      fieldName := getAttr(node, FieldRefAttr)
      hasFieldName := (fieldName != "")
      if hasFieldName {
        v.domConstruction.append("this.").append(fieldName).append("=")
      }

      // Construct raw elements differently from nested tomato templates
      if tagName == "tomato" {
        v.ignoreSubtree = true // Nested tomatos can't have children.

        src := getAttr(node, "src")
        if src == "" {
          return errors.New("Tomato element with no 'src' attribute!")
        }
        viewName := getViewName(src)
        v.domConstruction.append("new ").append(viewName).append("()")
        if hasFieldName {
          v.refs.PushBack(fieldName + ": " + viewName)
        }
      } else {
        v.domConstruction.append("Q.createOne('").append(tagName).append("')")
        if hasFieldName {
          v.refs.PushBack(fieldName + ": Q.OfOne")
        }
      }
    }

    // For all elements, we transfer any attributes set in the template
    v.transferAttrs(node)

  case html.TextNode:
    // Skip trailing whitespace nodes.
    if "" != strings.TrimSpace(node.Data) {
      v.domConstruction.append(".appendText(").append(escapeText(node.Data)).append(")")
    }
  }

  return nil // no error
}

// DF popping back up the stack.
func (v *typeScriptVisitor) tail(node *html.Node, depth int) {
  if v.appendStack.Len() > 0 && v.appendStack.Back().Value.(*html.Node) == node {
    v.appendStack.Remove(v.appendStack.Back())
    v.domConstruction.append(")")
    v.ignoreSubtree = false
  }
}

func (v *typeScriptVisitor) getView() string {
  return v.output.buffer.String()
}

func (v *typeScriptVisitor) emitPreamble() {
  v.output.append("\nexport class ").append(v.viewName).append(" extends Q.OfOne {")
}

func (v *typeScriptVisitor) emitElementRefs() {
  for e := v.refs.Front(); e != nil; e = e.Next() {
    fieldDecl := e.Value.(string)
    v.output.append("\n  ").append(fieldDecl).append(";")
    if e == v.refs.Back() {
      v.output.append("\n  ")
    }
  }
}

func (v *typeScriptVisitor) emitDomConstruction() {
  v.output.append("\n  constructor() {")
  v.output.append(v.domConstruction.buffer.String())
  v.output.append(";\n  }")
}

func (v *typeScriptVisitor) emitPostamble() {
  v.output.append("\n}\n")
}

func (v *typeScriptVisitor) transferAttrs(node *html.Node) {
  for _, attr := range node.Attr {

    // Skip _ref, _ignoreContent and src on a tomato
    if contains(blockedAttrs, attr.Key) || (strings.ToLower(node.Data) == "tomato" && attr.Key == "src") {
      continue
    }

    // Transform _id to id in the generated view.
    key := attr.Key
    if TunnelledIdAttr == attr.Key {
      key = IdAttr
    }
    emitAttr(&v.domConstruction, key, attr.Val)
  }
}

////////////////////////
// private functions
////////////////////////
func generateView(v viewGenerator) string {
  v.emitPreamble()
  v.emitElementRefs()
  v.emitDomConstruction()
  v.emitPostamble()
  return v.getView()
}

func escapeText(text string) string {
  return "'" + strings.Replace(text, "'", "\\'", -1) + "'"
}

func emitAttr(builder *stringBuilder, key, val string) {
  builder.append(".setAttr('").append(key).append("', '").append(val).append("')")
}

func contains(arr []string, val string) bool {
  for _, item := range arr {
    if item == val {
      return true
    }
  }
  return false
}

func hasAttr(node *html.Node, attr string) bool {
  return getAttr(node, attr) != ""
}

func getAttr(node *html.Node, attr string) string {
  for _, item := range node.Attr {
    if item.Key == attr {
      return item.Val
    }
  }
  return ""
}

func walk(fileName string, visitor viewGenerator) error {
  // open input file
  fi, err := os.Open(fileName)
  if err != nil {
    return err
  }

  // close fi on exit and check for its returned error
  defer func() {
    if err := fi.Close(); err != nil {
      fmt.Println(err.Error())
      // panic(err)
    }
  }()

  r := bufio.NewReader(fi)
  doc, err := html.Parse(r)
  if err != nil {
    return err
  }

  // Depth First traversal. Call the visitor going down the stack, and popping back up.
  var traverse func(n *html.Node, depth int) error
  traverse = func(n *html.Node, depth int) error {
    if n == nil {
      return errors.New("Template is empty!")
    }

    if err := visitor.head(n, depth); err != nil {
      return err
    }

    for c := n.FirstChild; c != nil; c = c.NextSibling {
      if err := traverse(c, depth+1); err != nil {
        return err
      }
    }

    visitor.tail(n, depth)
    return nil
  }

  // This Parser returns a well formed document. We only want to start our visitor on the
  // first child of the <body>. So let's find it!
  var findRoot func(n *html.Node) *html.Node
  findRoot = func(n *html.Node) *html.Node {
    if n.Data == "body" {
      return n.FirstChild
    }

    for c := n.FirstChild; c != nil; c = c.NextSibling {
      if rootElem := findRoot(c); rootElem != nil {
        return rootElem
      }
    }

    // Zilch
    return nil
  }

  rootElem := findRoot(doc)
  return traverse(rootElem, 0)
}

func indent(depth int) string {
  indent := "  "
  for i := 0; i < depth; i++ {
    indent += "  "
  }
  return "\n  " + indent
}

// Maps a file name to a class name for a generated View.
func getViewName(fileName string) string {
  slashStart := strings.LastIndex(fileName, "/")
  if slashStart < 0 {
    slashStart = 0
  } else {
    slashStart = slashStart + 1
  }

  viewName := fileName[slashStart:len(fileName)]
  viewName = strings.Replace(viewName, ".htmto", "", 1) + "View"
  return strings.ToUpper(viewName[0:1]) + viewName[1:len(viewName)]
}

func debugIdFromViewName(viewName string) string {
  return viewName[0 : len(viewName)-len("View")]
}
