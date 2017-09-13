/*
SPDX-License-Identifier: MIT

Copyright (c) 2017 Thanh Ha

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

// lhc is a checker to find code files missing license headers.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/zxiiro/license-header-checker/license"
)

var LICENSE_HEADER_LINES = 50
var VERSION = "0.1.0"

type License struct {
	Name string
	Text string
}

// Compare a license header with an approved list of license headers.
// Returns the name of the license that was approved. Else "".
func accepted_license(check string, approved []License) string {
	for _, i := range approved {
		if check == i.Text {
			return i.Name
		}
	}

	return ""
}

// check and exit if error.
func check(e error) {
	if e != nil {
		fmt.Println(e)
		os.Exit(1)
	}
}

func checkSPDX(license string, filename string) bool {
	file, err := os.Open(filename)
	check(err)
	defer file.Close()
	scanner := bufio.NewScanner(file)

	i := 0
	for scanner.Scan() {
		// Read only the first few lines to not read entire code file
		i++
		if i > LICENSE_HEADER_LINES {
			break
		}

		s := strings.ToUpper(scanner.Text())
		if strings.Contains(s, "SPDX-LICENSE-IDENTIFIER:") {
			spdx := stripSpaces(strings.SplitN(s, ":", 2)[1])
			if spdx == license {
				return true
			} else {
				return false
			}
		}
	}

	return false
}

func findFiles(directory string, patterns []string) []string {
	var files []string
	filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			for _, p := range patterns {
				f, _ := filepath.Glob(filepath.Join(path, p))
				files = append(files, f...)
			}
		}
		return nil
	})
	return files
}

// fetchLicense from file and return license text.
func fetchLicense(filename string) string {
	comment, multilineComment := false, false
	licenseText := ""

	var scanner *bufio.Scanner
	if filename == "MIT" {
		scanner = bufio.NewScanner(strings.NewReader(license.MIT_LICENSE))
	} else if filename == "EPL-1.0" {
		scanner = bufio.NewScanner(strings.NewReader(license.EPL_10_LICENSE))
	} else {
		file, err := os.Open(filename)
		check(err)
		defer file.Close()

		// Read the first 2 bytes to decide if it is a comment string
		b := make([]byte, 2)
		_, err = file.Read(b)
		check(err)
		if isComment(string(b)) {
			comment = true
		}
		file.Seek(0, 0) // Reset so we can read the full file next

		scanner = bufio.NewScanner(file)
	}

	i := 0
	for scanner.Scan() {
		// Read only the first few lines to not read entire code file
		i++
		if i > LICENSE_HEADER_LINES {
			break
		}

		s := scanner.Text()

		if ignoreComment(s) {
			continue
		}

		if comment == true {
			if strings.HasPrefix(s, "/*") {
				multilineComment = true
			} else if strings.Contains(s, "*/") {
				multilineComment = false
			}

			if !multilineComment && !isComment(s) ||
				// EPL headers can contain contributors list.
				strings.Contains(strings.ToUpper(s), " * CONTRIBUTORS:") {
				break
			}

			s = trimComment(s)
		}

		licenseText += s
	}

	return stripSpaces(licenseText)
}

// Check if a string is a comment line.
func isComment(str string) bool {
	if !strings.HasPrefix(str, "#") &&
		!strings.HasPrefix(str, "//") &&
		!strings.HasPrefix(str, "/*") {
		return false
	}

	return true
}

// Ignore certain lines containing key strings
func ignoreComment(str string) bool {
	s := strings.ToUpper(str)
	if strings.HasPrefix(s, "#!") ||
		strings.Contains(s, "COPYRIGHT") ||
		strings.Contains(s, "SPDX-LICENSE-IDENTIFIER") ||
		// License name in LICENSE file but not header
		strings.Contains(s, "MIT LICENSE") {
		return true
	}

	return false
}

// Strip whitespace from string.
func stripSpaces(str string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, str)
}

// Trim the comment prefix from string.
func trimComment(str string) string {
	str = strings.TrimLeft(str, "#")
	str = strings.TrimLeft(str, "//")
	str = strings.TrimLeft(str, "/*")
	str = strings.TrimLeft(str, " *")
	str = strings.Split(str, "*/")[0]
	str = strings.TrimLeft(str, "*")
	return str
}

// Usage prints a statement to explain how to use this command.
func usage() {
	fmt.Printf("Usage: %s [OPTIONS] [FILE]...\n", os.Args[0])
	fmt.Printf("Compare FILE with an expected license header.\n")
	fmt.Printf("\nOptions:\n")
	flag.PrintDefaults()
}

func main() {
	directoryPtr := flag.String("directory", ".", "Directory to search for files.")
	SPDXPtr := flag.Bool("spdx", false, "Verify SDPX identifier matches license.")
	licensePtr := flag.String("license", "license.txt", "Comma-separated list of license files to compare against.")
	versionPtr := flag.Bool("version", false, "Print version")

	flag.Usage = usage
	flag.Parse()

	if *versionPtr {
		fmt.Println("License Checker version", VERSION)
		os.Exit(0)
	}

	fmt.Println("Search Patterns:", flag.Args())

	var accepted_licenses []License
	for _, l := range strings.Split(*licensePtr, ",") {
		license := License{l, fetchLicense(l)}
		accepted_licenses = append(accepted_licenses, license)
	}
	checkFiles := findFiles(*directoryPtr, flag.Args())

	for _, file := range checkFiles {
		headerText := fetchLicense(file)
		license := accepted_license(headerText, accepted_licenses)
		result := ""
		if license != "" {
			result = result + "✔"
		} else {
			result = result + "✘"
		}
		if *SPDXPtr {
			if checkSPDX(license, file) {
				result = result + "✔"
			} else {
				result = result + "✘"
			}
		}
		fmt.Println(result, file)
	}
}
