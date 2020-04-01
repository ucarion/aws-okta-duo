package provider

import "golang.org/x/net/html"

func findNodeByName(node *html.Node, name string) *html.Node {
	for _, attr := range node.Attr {
		if attr.Key == "name" && attr.Val == name {
			return node
		}
	}

	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if node := findNodeByName(c, name); node != nil {
			return node
		}
	}

	return nil
}
