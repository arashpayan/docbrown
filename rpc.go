package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/russross/blackfriday"
)

// RPCDoc ...
type RPCDoc struct {
	Command     string      `json:"command"`
	Description string      `json:"description,omitempty"`
	Samples     []DocSample `json:"samples,omitempty"`
	PackageName string      `json:"package_name"`
}

// HTMLDescription converts the description from markdown to html
func (rd RPCDoc) HTMLDescription() string {
	return string(blackfriday.MarkdownCommon([]byte(rd.Description)))
}

// HTMLID returns an id capable of being used in an HTML document
func (rd RPCDoc) HTMLID() string {
	return fmt.Sprintf("command_%s", strings.ToLower(rd.Command))
}

func (rd RPCDoc) String() string {
	buf, err := json.MarshalIndent(rd, "", "  ")
	if err != nil {
		return "<nil>"
	}

	return string(buf)
}

type byRPCCommand []*RPCDoc

func (a byRPCCommand) Len() int           { return len(a) }
func (a byRPCCommand) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byRPCCommand) Less(i, j int) bool { return strings.Compare(a[i].Command, a[j].Command) < 0 }
