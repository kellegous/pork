package pork

import (
  "bytes"
  "container/list"
  "io/ioutil"
  "os"
  "path/filepath"
  "strings"
)

const (
  tomatoFileExtension = ".htmto"
)

func GenerateTomatoes(root string, outFile string, language Language, qImport string, forceDebugIds bool) error {
  files, err := collectTomatoFiles(root)
  if err != nil {
    return err
  }

  generator, err := MakeTomatoGenerator(root, language, qImport)
  if err != nil {
    return err
  }

  // Now that we have the tomato file paths. Go ahead and generate the view strings.
  views, err := generator.GenerateViews(files, forceDebugIds)
  if err != nil {
    return err
  }

  // Write the file to disk.
  if err := writeTomatoOutput(outFile, views, generator); err != nil {
    return err
  }

  return nil
}

func collectTomatoFiles(root string) (*list.List, error) {
  l := list.New()
  err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
    if err != nil {
      return err
    } else if !info.IsDir() && strings.HasSuffix(info.Name(), tomatoFileExtension) {
      l.PushBack(path)
    }
    return nil
  })

  if err != nil {
    return nil, err
  } else {
    return l, nil
  }
}

// Write the generated views to a file. This file should never ever be more than
// on the order of a few thousand lines, so it lives all in memory.
func writeTomatoOutput(outFile string, views map[string]string, generator TomatoGenerator) error {
  buffer := &bytes.Buffer{}

  generator.EmitPreamble(buffer)
  for _, content := range views {
    buffer.WriteString(content)
    buffer.WriteString("\n\n")
  }
  generator.EmitPostamble(buffer)

  // Dump the file to disk.
  if err := os.MkdirAll(filepath.Dir(outFile), 0777); err != nil {
    return err
  }
  return ioutil.WriteFile(outFile, buffer.Bytes(), 0644)
}
