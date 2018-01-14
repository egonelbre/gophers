// fix-svg-style fixes Inkscape palette to be compatible with Affinity Designer
//

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func main() {
	err := filepath.Walk("vector", func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".svg" {
			return ProcessSVGFile(path)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
}

func ProcessSVGFile(file string) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	input := bytes.NewReader(data)
	context := &html.Node{
		Type: html.ElementNode,
	}

	nodes, err := html.ParseFragment(input, context)
	if err != nil {
		return err
	}

	changed := ProcessSVG(nodes)

	output := bytes.NewBuffer(nil)
	for _, node := range nodes {
		html.Render(output, node)
	}

	outdata := output.Bytes()
	if changed {
		fmt.Println("Wrinting ", file)
		err2 := ioutil.WriteFile(file, outdata, 0755)
		if err2 != nil {
			return err2
		}
	}

	return nil
}

func ProcessSVG(nodes []*html.Node) bool {
	changed := false

	getElementByID := map[string]*html.Node{}
	var process func(node *html.Node)

	rxRemoveStyle := regexp.MustCompile(`(visibility:visible)[;$]`)
	rxFillStyle := regexp.MustCompile(`fill:url\((#[a-zA-Z0-9\-]+)\)`)
	rxStrokeStyle := regexp.MustCompile(`stroke:url\((#[a-zA-Z0-9\-]+)\)`)
	rxStopColor := regexp.MustCompile(`stop-color:([^;]*)[;$]`)

	resolveColorCache := map[string]string{}
	resolveColor := func(id string) string {
		if cached, ok := resolveColorCache[id]; ok {
			return cached
		}

		node := getElementByID[id]
		for {
			xlink := GetAttributeValue(node, "href")
			if xlink == "" {
				break
			}
			node = getElementByID[xlink[1:]]
		}

		stops := GetElementsByTagName(node, "stop")
		if len(stops) == 0 {
			return "rgba(0,0,0,0)"
		}
		if len(stops) >= 2 {
			return "url(#" + id + ")"
		}

		stop := stops[0]
		style := GetAttributeValue(stop, "style")
		match := rxStopColor.FindStringSubmatch(style)

		resolveColorCache[id] = match[1]
		return match[1]
	}

	process = func(node *html.Node) {
		// build index
		id := GetAttributeValue(node, "id")
		if id != "" {
			getElementByID[id] = node
		}

		// recurse
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			process(child)
		}

		if style := GetAttribute(node, "style"); style != nil {
			style.Val = rxRemoveStyle.ReplaceAllString(style.Val, "")
			style.Val = rxFillStyle.ReplaceAllStringFunc(style.Val, func(attribute string) string {
				// fill:url(#xyz)
				colorId := attribute[10 : len(attribute)-1]

				replacement := "fill:" + resolveColor(colorId)
				changed = changed || (replacement != attribute)
				return replacement
			})

			style.Val = rxStrokeStyle.ReplaceAllStringFunc(style.Val, func(attribute string) string {
				// colore:url(#xyz)
				colorId := attribute[12 : len(attribute)-1]

				replacement := "stroke:" + resolveColor(colorId)
				changed = changed || (replacement != attribute)
				return replacement
			})
		}
	}

	for _, node := range nodes {
		process(node)
	}

	// remove all dead gradients
	for _, node := range nodes {
		defss := GetElementsByTagName(node, "defs")
		for _, defs := range defss {
			toRemove := []*html.Node{}
			for child := defs.FirstChild; child != nil; child = child.NextSibling {
				if child.Data == "linearGradient" {
					id := GetAttributeValue(child, "id")
					noReplacement := "url(#" + id + ")"
					if noReplacement != resolveColor(id) {
						toRemove = append(toRemove, child)
					}
				}
			}

			for _, child := range toRemove {
				changed = true
				defs.RemoveChild(child)
			}
		}
	}

	return changed
}

func GetAttributeValue(node *html.Node, name string) string {
	for i := range node.Attr {
		if node.Attr[i].Key == name {
			return node.Attr[i].Val
		}
	}
	return ""
}

func GetAttribute(node *html.Node, name string) *html.Attribute {
	for i := range node.Attr {
		if node.Attr[i].Key == name {
			return &node.Attr[i]
		}
	}
	return nil
}

func GetElementsByTagName(node *html.Node, tagname string) []*html.Node {
	xs := []*html.Node{}

	dataAtom := atom.Lookup([]byte(tagname))
	if dataAtom != 0 {
		tagname = ""
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != html.ElementNode {
			continue
		}
		if child.DataAtom == dataAtom && child.Data == tagname {
			xs = append(xs, child)
			continue
		}
	}
	return xs
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
