package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/russross/blackfriday"
)

// BroadcastDoc ...
type BroadcastDoc struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Samples     []DocSample `json:"samples,omitempty"`
	PackageName string      `json:"package_name"`
}

// HTMLDescription converts the description from markdown to html
func (bd BroadcastDoc) HTMLDescription() string {
	return string(blackfriday.MarkdownCommon([]byte(bd.Description)))
}

// HTMLID returns an id capable of being used in an HTML document
func (bd BroadcastDoc) HTMLID() string {
	return fmt.Sprintf("broadcast_%s", strings.ToLower(bd.Name))
}

func (bd BroadcastDoc) String() string {
	buf, err := json.MarshalIndent(bd, "", "  ")
	if err != nil {
		return "<nil>"
	}

	return string(buf)
}

type byBroadcastName []*BroadcastDoc

func (a byBroadcastName) Len() int           { return len(a) }
func (a byBroadcastName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byBroadcastName) Less(i, j int) bool { return strings.Compare(a[i].Name, a[j].Name) < 0 }
