// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package loglineparser

import (
	"iter"
	"regexp"
	"strings"
)

// Position represents the start and end positions of content in a string
type Position struct {
	Start int
	End   int
}

// Node represents a fundamental unit of parsed log data in a doubly linked list
type Node struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	Label   string `json:"label,omitempty"`
	Prev    *Node  `json:"-"`
	Next    *Node  `json:"-"`
}

// Node type constants
const (
	NodeTypeText      = "text"
	NodeTypeAnsi      = "ansi"
	NodeTypeJSON      = "json"
	NodeTypeString    = "string"
	NodeTypeUUID      = "uuid"
	NodeTypeTimestamp = "timestamp"
	NodeTypeURL       = "url"
	NodeTypeNumber    = "number"
)

// SetLabel sets the label for a node
func (n *Node) SetLabel(label string) {
	n.Label = label
}

// IsTextType returns true if the node is a text type
func (n *Node) IsTextType() bool {
	return n.Type == NodeTypeText
}

// IsAnsiType returns true if the node is an ANSI type
func (n *Node) IsAnsiType() bool {
	return n.Type == NodeTypeAnsi
}

// IsStructuredType returns true if the node is a structured type (JSON, string, UUID, etc.)
func (n *Node) IsStructuredType() bool {
	return n.Type == NodeTypeJSON || n.Type == NodeTypeString || n.Type == NodeTypeUUID ||
		n.Type == NodeTypeTimestamp || n.Type == NodeTypeURL || n.Type == NodeTypeNumber
}

// Splice replaces the current node with a slice of new nodes and returns (head, next)
func (n *Node) Splice(newNodes []Node) (*Node, *Node) {
	nextNode := n.Next

	if len(newNodes) == 0 {
		// Remove the current node
		if n.Prev != nil {
			n.Prev.Next = n.Next
		}
		if n.Next != nil {
			n.Next.Prev = n.Prev
		}
		return nextNode, nextNode
	}

	// Create linked list from slice
	var head *Node
	var prev *Node
	for i := range newNodes {
		node := &newNodes[i]
		if head == nil {
			head = node
		}
		node.Prev = prev
		if prev != nil {
			prev.Next = node
		}
		prev = node
	}

	// Connect the new linked list to the existing chain
	if n.Prev != nil {
		n.Prev.Next = head
		head.Prev = n.Prev
	}
	if n.Next != nil {
		prev.Next = n.Next
		n.Next.Prev = prev
	}

	return head, nextNode
}

// All returns an iterator that yields all nodes starting from this node
func (n *Node) All() iter.Seq[*Node] {
	return func(yield func(*Node) bool) {
		current := n
		for current != nil {
			if !yield(current) {
				return
			}
			current = current.Next
		}
	}
}

// processAnsiEscapes takes a text node and splits it into alternating text and ansi nodes
func processAnsiEscapes(node Node) []Node {
	if node.Type != NodeTypeText {
		return []Node{node}
	}

	// Fast path: if no ANSI escapes are found, return original node
	if !strings.Contains(node.Content, "\x1b[") {
		return []Node{node}
	}

	var result []Node
	content := node.Content
	lastIndex := 0

	// ANSI escape sequence regex
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	matches := ansiRegex.FindAllStringIndex(content, -1)

	for _, match := range matches {
		matchStart := match[0]
		matchEnd := match[1]

		// Add text before this ANSI sequence
		if matchStart > lastIndex {
			textContent := content[lastIndex:matchStart]
			result = append(result, Node{Type: NodeTypeText, Content: textContent})
		}

		// Add the ANSI sequence itself
		ansiContent := content[matchStart:matchEnd]
		result = append(result, Node{Type: NodeTypeAnsi, Content: ansiContent})

		lastIndex = matchEnd
	}

	// Add remaining text after last ANSI sequence
	if lastIndex < len(content) {
		textContent := content[lastIndex:]
		result = append(result, Node{Type: NodeTypeText, Content: textContent})
	}

	return result
}

// ParseLineToNode creates an initial node containing the full log line as text
func ParseLineToNode(line string) *Node {
	return &Node{Type: NodeTypeText, Content: line}
}
