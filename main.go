package main

import (
	"bytes"
	"flag"
	"go/format"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"unicode"

	"gopkg.in/yaml.v2"
)

// commonInitialisms is a set of common initialisms.
// Only add entries that are highly unlikely to be non-initialisms.
// For instance, "ID" is fine (Freudian code is rare), but "AND" is not.
// https://github.com/golang/lint/blob/master/lint.go
var commonInitialisms = map[string]bool{
	"API":   true,
	"ASCII": true,
	"CPU":   true,
	"CSS":   true,
	"DNS":   true,
	"EOF":   true,
	"GUID":  true,
	"HTML":  true,
	"HTTP":  true,
	"HTTPS": true,
	"ID":    true,
	"IP":    true,
	"JSON":  true,
	"LHS":   true,
	"QPS":   true,
	"RAM":   true,
	"RHS":   true,
	"RPC":   true,
	"SLA":   true,
	"SMTP":  true,
	"SQL":   true,
	"SSH":   true,
	"TCP":   true,
	"TLS":   true,
	"TTL":   true,
	"UDP":   true,
	"UI":    true,
	"UID":   true,
	"UUID":  true,
	"URI":   true,
	"URL":   true,
	"UTF8":  true,
	"VM":    true,
	"XML":   true,
	"XSRF":  true,
	"XSS":   true,
}

var progName = filepath.Base(os.Args[0])

var (
	inputFile  = flag.String("i", "", "input yaml file")
	outputFile = flag.String("o", "", "output go file, output to stdout if not specified")
	pkg        = flag.String("p", "main", "package to generate")
	prefix     = flag.String("r", "", "prefix for constants")
)

func main() {
	flag.Parse()
	if *inputFile == "" {
		flag.Usage()
		return
	}

	bb, err := ioutil.ReadFile(*inputFile)
	if err != nil {
		log.Fatalf("File reading error: %s", err)
	}

	data, err := generate(bb, *pkg, *prefix)
	if err != nil {
		log.Fatalf("Error while generating: %s", err)
	}

	if *outputFile == "" {
		os.Stdout.Write(data) // nolint
		return
	}
	err = ioutil.WriteFile(*outputFile, data, 0644)
	if err != nil {
		log.Fatalf("Error while writing result file: %s", err)
	}
}

func generate(srcData []byte, pkg, prefix string) ([]byte, error) {

	srcMap := make(map[string]interface{})
	err := yaml.Unmarshal(srcData, srcMap)
	if err != nil {
		return nil, err
	}

	flMap := Flatten(srcMap)
	keys := make([]string, 0, len(flMap))

	for k := range flMap {
		keys = append(keys, k)
		flMap[k] = toCamelCase(k)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return strings.Compare(keys[i], keys[j]) < 0
	})

	tmpl, err := template.New("constTmpl").Parse(constTmpl)
	if err != nil {
		return nil, err
	}

	var buffer = new(bytes.Buffer)
	dot := map[string]interface{}{
		"pkg":    pkg,
		"prefix": prefix,
		"header": "generated by " + progName,
		"keys":   keys,
		"map":    flMap,
	}
	err = tmpl.Execute(buffer, dot)
	if err != nil {
		return nil, err
	}

	data, err := format.Source(buffer.Bytes())
	if err != nil {
		return buffer.Bytes(), err
	}
	return data, err
}

func toCamelCase(id string) string {
	var result string

	r := regexp.MustCompile(`[\-\.\_\s]`)
	words := r.Split(id, -1)

	for _, w := range words {
		upper := strings.ToUpper(w)
		if commonInitialisms[upper] {
			result += upper
			continue
		}

		if len(w) > 0 {
			u := []rune(w)
			u[0] = unicode.ToUpper(u[0])
			result += string(u)
		}
	}
	return result
}
