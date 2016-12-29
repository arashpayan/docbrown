package main

import (
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"unicode"

	"github.com/russross/blackfriday"
)

type sampleType int

const (
	sampleTypeBody     sampleType = 0
	sampleTypeResponse            = 1
)

// DocSample ...
type DocSample struct {
	Text     string     `json:"text"`
	Code     string     `json:"code"`
	Type     sampleType `json:"type"`
	Language string     `json:"language"`
}

// HTMLText returns an HTML representation of the Text
func (ds DocSample) HTMLText() string {
	return string(blackfriday.MarkdownCommon([]byte(ds.Text)))
}

func (ds DocSample) String() string {
	buf, err := json.MarshalIndent(ds, "", "  ")
	if err != nil {
		return "<nil>"
	}

	return string(buf)
}

// EndpointArgument ...
type EndpointArgument struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// RESTDoc ...
type RESTDoc struct {
	Endpoint    string             `json:"endpoint"`
	Method      string             `json:"method,omitempty"`
	Description string             `json:"description,omitempty"`
	Samples     []DocSample        `json:"samples,omitempty"`
	PackageName string             `json:"package_name"`
	PathArgs    []EndpointArgument `json:"path_arguments,omitempty"`
	QueryArgs   []EndpointArgument `json:"query_argument,omitempty"`
	Purpose     string             `json:"purpose,omitempty"`
}

// HTMLDescription converts the description from markdown to html
func (rd RESTDoc) HTMLDescription() string {
	return string(blackfriday.MarkdownCommon([]byte(rd.Description)))
}

// LowercaseMethod returns the HTTP method as a lowercase string
func (rd RESTDoc) LowercaseMethod() string {
	return strings.ToLower(rd.Method)
}

// HTMLID returns an id capable of being used in an HTML document
func (rd RESTDoc) HTMLID() string {
	epParts := strings.FieldsFunc(rd.Endpoint, func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c)
	})
	return fmt.Sprintf("%s_%s", strings.Join(epParts, "_"), strings.ToLower(rd.Method))
}

func (rd RESTDoc) String() string {
	buf, err := json.MarshalIndent(rd, "", "  ")
	if err != nil {
		return "<nil>"
	}

	return string(buf)
}

// PackageDoc ...
type PackageDoc struct {
	Name          string
	Description   string
	RESTDocs      []*RESTDoc
	RPCDocs       []*RPCDoc
	BroadcastDocs []*BroadcastDoc
}

var endpointRE = regexp.MustCompile(`@endpoint +(\S+)`)
var methodRE = regexp.MustCompile("@method +(DELETE|GET|POST|PUT)")
var commandRE = regexp.MustCompile("@command +([^@]*)")
var sampleRE = regexp.MustCompile(`(@sampleBody|@sampleResponse)[^@]*`)
var samplePartsRE = regexp.MustCompile("(@sampleBody|@sampleResponse) *\\n((.|\\n)+)?``` *(.*)((.|\\n)+)(?:```)")
var descriptionRE = regexp.MustCompile(`@description +([^@]+)`)
var broadcastRE = regexp.MustCompile(`@broadcast +(\w+)`)
var packageRE = regexp.MustCompile(`@package +(\S+)`)
var pathArgRE = regexp.MustCompile(`@pathArg +(\w*) +(.*)`)
var purposeRE = regexp.MustCompile(`@purpose +(.+)`)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("You need to specify a path to scan")
	}
	if len(os.Args) < 3 {
		log.Fatal("You need to specify a path for the output directory")
	}
	pkgs, err := parser.ParseDir(token.NewFileSet(), os.Args[1], nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	pkgDocs := make(map[string]*PackageDoc)

	for _, pkg := range pkgs {
		for _, srcFile := range pkg.Files {
			for _, cg := range srcFile.Comments {
				for _, cmnt := range cg.List {
					comment := cmnt.Text
					packageMatches := packageRE.FindStringSubmatch(comment)
					if packageMatches == nil {
						continue
					}
					if rd := parseRESTDoc(comment); rd != nil {
						pkgDoc := pkgDocs[rd.PackageName]
						if pkgDoc == nil {
							pkgDoc = &PackageDoc{Name: rd.PackageName}
							pkgDocs[pkgDoc.Name] = pkgDoc
						}
						pkgDoc.RESTDocs = append(pkgDoc.RESTDocs, rd)
						continue
					}
					if rpcDoc := parseRPCDoc(comment); rpcDoc != nil {
						pkgDoc := pkgDocs[rpcDoc.PackageName]
						if pkgDoc == nil {
							pkgDoc = &PackageDoc{Name: rpcDoc.PackageName}
							pkgDocs[pkgDoc.Name] = pkgDoc
						}
						pkgDoc.RPCDocs = append(pkgDoc.RPCDocs, rpcDoc)
						continue
					}
					if bcastDoc := parseBroadcastDoc(comment); bcastDoc != nil {
						pkgDoc := pkgDocs[bcastDoc.PackageName]
						if pkgDoc == nil {
							pkgDoc = &PackageDoc{Name: bcastDoc.PackageName}
							pkgDocs[pkgDoc.Name] = pkgDoc
						}
						pkgDoc.BroadcastDocs = append(pkgDoc.BroadcastDocs, bcastDoc)
						continue
					}
				}
			}
		}
	}

	// sort the commands and broadcasts for each package
	for _, pkgDoc := range pkgDocs {
		sortableRPCs := byRPCCommand(pkgDoc.RPCDocs)
		sort.Sort(sortableRPCs)

		sortableBCs := byBroadcastName(pkgDoc.BroadcastDocs)
		sort.Sort(sortableBCs)
	}

	var allPkgNames []string
	for pkgName := range pkgDocs {
		allPkgNames = append(allPkgNames, pkgName)
	}
	sort.Strings(allPkgNames)
	tmpl, err := template.ParseFiles("package_template.html")
	if err != nil {
		log.Fatal(err)
	}

	outputDir := filepath.Join(os.Args[2], "docs")
	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		log.Fatalf("Error creating output dir: %v", err)
	}

	for pkgName, pkgDoc := range pkgDocs {
		fileName := filepath.Join(outputDir, pkgName+".html")
		file, err := os.Create(fileName)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		err = tmpl.Execute(file, map[string]interface{}{
			"PackageNames": allPkgNames,
			"PackageDocs":  pkgDoc,
		})
		if err != nil {
			log.Fatalf("Template error: %v", err)
		}
	}

	// create the index file
	tmpl, err = template.ParseFiles("index_template.html")
	if err != nil {
		log.Fatal(err)
	}

	fileName := filepath.Join(outputDir, "index.html")
	file, err := os.Create(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	err = tmpl.Execute(file, map[string]interface{}{
		"PackageNames": allPkgNames,
	})
	if err != nil {
		log.Fatalf("Template error: %v", err)
	}

	// copy the stylesheet and js files
	err = copyFile("prism.css", filepath.Join(outputDir, "prism.css"))
	if err != nil {
		log.Fatalf("Error copying prism.css: %v", err)
	}
	err = copyFile("prism.js", filepath.Join(outputDir, "prism.js"))
	if err != nil {
		log.Fatalf("Error copying prism.js: %v", err)
	}
	err = copyFile("style.css", filepath.Join(outputDir, "style.css"))
	if err != nil {
		log.Fatalf("Error copying docs.css: %v", err)
	}
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("error opening src: %v", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("error opening dst: %v", err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("error copying data: %v", err)
	}

	return nil
}

func parseRESTDoc(comment string) *RESTDoc {
	pkgs := packageRE.FindStringSubmatch(comment)
	if len(pkgs) < 2 {
		return nil
	}
	endpoints := endpointRE.FindStringSubmatch(comment)
	if len(endpoints) < 2 {
		return nil
	}

	doc := RESTDoc{PackageName: pkgs[1], Endpoint: endpoints[1]}

	// default method is GET
	methods := methodRE.FindStringSubmatch(comment)
	if methods == nil {
		doc.Method = "GET"
	} else {
		doc.Method = methods[1]
	}

	doc.Samples = parseDocSamples(comment)

	descriptionMatches := descriptionRE.FindStringSubmatch(comment)
	if len(descriptionMatches) > 1 {
		doc.Description = strings.TrimSpace(descriptionMatches[1])
	}

	// look for path arguments
	pathArgs := pathArgRE.FindAllStringSubmatch(comment, -1)
	if len(descriptionMatches) > 0 {
		for _, paParts := range pathArgs {
			pa := EndpointArgument{
				Name:        paParts[1],
				Description: paParts[2],
			}
			doc.PathArgs = append(doc.PathArgs, pa)
		}
	}

	purposeMatches := purposeRE.FindStringSubmatch(comment)
	if purposeMatches != nil {
		doc.Purpose = purposeMatches[1]
	}

	return &doc
}

func parseRPCDoc(comment string) *RPCDoc {
	pkgs := packageRE.FindStringSubmatch(comment)
	if len(pkgs) < 2 {
		return nil
	}
	commandMatches := commandRE.FindStringSubmatch(comment)
	if commandMatches == nil {
		return nil
	}

	doc := RPCDoc{PackageName: pkgs[1], Command: strings.TrimSpace(commandMatches[1])}

	doc.Samples = parseDocSamples(comment)

	descriptionMatches := descriptionRE.FindStringSubmatch(comment)
	if descriptionMatches != nil {
		doc.Description = descriptionMatches[1]
	}

	return &doc
}

func parseBroadcastDoc(comment string) *BroadcastDoc {
	packageMatches := packageRE.FindStringSubmatch(comment)
	if packageMatches == nil {
		return nil
	}
	bcastMatches := broadcastRE.FindStringSubmatch(comment)
	if bcastMatches == nil {
		return nil
	}

	doc := BroadcastDoc{PackageName: packageMatches[1], Name: bcastMatches[1]}

	doc.Samples = parseDocSamples(comment)

	descriptionMatches := descriptionRE.FindStringSubmatch(comment)
	if descriptionMatches != nil {
		doc.Description = descriptionMatches[1]
	}

	return &doc
}

func parseDocSamples(comment string) []DocSample {
	var samples []DocSample

	sampleMatches := sampleRE.FindAllString(comment, -1)
	for _, sample := range sampleMatches {
		parts := samplePartsRE.FindStringSubmatch(sample)
		if len(parts) == 0 {
			continue
		}
		ds := DocSample{}
		if parts[1] == "@sampleBody" {
			ds.Type = sampleTypeBody
		} else if parts[1] == "@sampleResponse" {
			ds.Type = sampleTypeResponse
		}
		ds.Text = parts[2]
		ds.Language = parts[4]
		ds.Code = strings.TrimSpace(parts[5])
		samples = append(samples, ds)
	}

	return samples
}
