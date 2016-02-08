package main

import (
	"encoding/json"
	"go/parser"
	"go/token"
	"log"
	"os"
	"regexp"
	"text/template"
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
}

func (rd RESTDoc) String() string {
	buf, err := json.MarshalIndent(rd, "", "  ")
	if err != nil {
		return "<nil>"
	}

	return string(buf)
}

// RPCDoc ...
type RPCDoc struct {
	Command     string      `json:"command"`
	Description string      `json:"description,omitempty"`
	Samples     []DocSample `json:"samples,omitempty"`
	PackageName string      `json:"package_name"`
}

func (rd RPCDoc) String() string {
	buf, err := json.MarshalIndent(rd, "", "  ")
	if err != nil {
		return "<nil>"
	}

	return string(buf)
}

// BroadcastDoc ...
type BroadcastDoc struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Samples     []DocSample `json:"samples,omitempty"`
	PackageName string      `json:"package_name"`
}

func (bd BroadcastDoc) String() string {
	buf, err := json.MarshalIndent(bd, "", "  ")
	if err != nil {
		return "<nil>"
	}

	return string(buf)
}

// PackageDoc ...
type PackageDoc struct {
	Name        string
	Description *string
}

var endpointRE = regexp.MustCompile(`@endpoint +(\S+)`)
var methodRE = regexp.MustCompile("@method +(DELETE|GET|POST|PUT)")
var commandRE = regexp.MustCompile("@command +([^@]*)")
var sampleRE = regexp.MustCompile(`(@sampleBody|@sampleResponse)[^@]*`)
var samplePartsRE = regexp.MustCompile("(@sampleBody|@sampleResponse) *\\n((.|\\n)+)?``` *(.*)((.|\\n)+)(?:```)")
var descriptionRE = regexp.MustCompile(`@description +([^@]+)`)
var broadcastRE = regexp.MustCompile(`@broadcast +(\w+)`)
var packageRE = regexp.MustCompile(`@package +(\w+)`)
var pathArgRE = regexp.MustCompile(`@pathArg +(\w*) +(.*)`)

// var purposeRE = regexp.MustCompile()

func main() {
	if len(os.Args) < 2 {
		log.Fatal("You need to specify a path to scan")
	}
	pkgs, err := parser.ParseDir(token.NewFileSet(), os.Args[1], nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	var restDocs []*RESTDoc
	var rpcDocs []*RPCDoc
	var broadcastDocs []*BroadcastDoc
	pkgDocLists := map[string][]interface{}{}

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
						restDocs = append(restDocs, rd)
						list := pkgDocLists[rd.PackageName]
						list = append(list, rd)
						pkgDocLists[rd.PackageName] = list
						continue
					}
					if rpcDoc := parseRPCDoc(comment); rpcDoc != nil {
						rpcDocs = append(rpcDocs, rpcDoc)
						list := pkgDocLists[rpcDoc.PackageName]
						list = append(list, rpcDoc)
						pkgDocLists[rpcDoc.PackageName] = list
						continue
					}
					if bcastDoc := parseBroadcastDoc(comment); bcastDoc != nil {
						broadcastDocs = append(broadcastDocs, bcastDoc)
						list := pkgDocLists[bcastDoc.PackageName]
						list = append(list, bcastDoc)
						pkgDocLists[bcastDoc.PackageName] = list
						continue
					}
				}
			}
		}
	}

	var allPkgNames []string
	for pkgName := range pkgDocLists {
		allPkgNames = append(allPkgNames, pkgName)
	}
	log.Printf("%v", allPkgNames)
	tmpl, err := template.ParseFiles("package_template.html")
	if err != nil {
		log.Fatal(err)
	}

	for pkgName := range pkgDocLists {
		file, err := os.Create(pkgName + ".html")
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		tmpl.Execute(file, map[string]interface{}{
			"PackageNames": allPkgNames,
		})
	}
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
	method := "GET"
	methods := methodRE.FindStringSubmatch(comment)
	if methods == nil {
		doc.Method = "GET"
	} else {
		doc.Method = method
	}

	doc.Samples = parseDocSamples(comment)

	descriptionMatches := descriptionRE.FindStringSubmatch(comment)
	if len(descriptionMatches) > 1 {
		doc.Description = descriptionMatches[1]
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

	return &doc
}

func parseRPCDoc(comment string) *RPCDoc {
	pkgs := packageRE.FindStringSubmatch(comment)
	if len(pkgs) < 2 {
		return nil
	}
	commandMatch := commandRE.FindStringSubmatch(comment)
	if commandMatch == nil {
		return nil
	}

	doc := RPCDoc{PackageName: pkgs[1]}

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
		ds.Code = parts[5]
		samples = append(samples, ds)
	}

	return samples
}
