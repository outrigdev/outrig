// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package loglineparser

import (
	"iter"
	"regexp"
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

var ansiEscapeRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)
var uuidRegex = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)

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
func (n *Node) Splice(newNodes []*Node) (*Node, *Node) {
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

	// Special case: if there's only one node and it's the same as the current node, no change needed
	if len(newNodes) == 1 && newNodes[0] == n {
		return n, n.Next
	}

	// Create linked list from slice
	head := newNodes[0]
	var prev *Node
	for _, node := range newNodes {
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

// FindFirstAnsiEscape finds the first ANSI escape sequence in the given string
// Returns nil if no ANSI escape is found, otherwise returns the position range
func FindFirstAnsiEscape(s string) *Position {
	if len(s) == 0 {
		return nil
	}

	match := ansiEscapeRegex.FindStringIndex(s)
	if match == nil {
		return nil
	}

	return &Position{Start: match[0], End: match[1]}
}

// FindFirstUUID finds the first UUID in the given string
// Returns nil if no UUID is found, otherwise returns the position range
func FindFirstUUID(s string) *Position {
	if len(s) == 0 {
		return nil
	}

	match := uuidRegex.FindStringIndex(s)
	if match == nil {
		return nil
	}

	return &Position{Start: match[0], End: match[1]}
}

// FindFirstURL finds the first HTTP/HTTPS URL in the given string
// Returns nil if no URL is found, otherwise returns the position range
// Applies heuristics to avoid including trailing punctuation
func FindFirstURL(s string) *Position {
	if len(s) == 0 {
		return nil
	}

	// Look for http:// or https://
	httpIndex := -1
	for i := 0; i <= len(s)-7; i++ {
		if i <= len(s)-8 && s[i:i+8] == "https://" {
			httpIndex = i
			break
		}
		if s[i:i+7] == "http://" {
			httpIndex = i
			break
		}
	}

	if httpIndex == -1 {
		return nil
	}

	// Find the end of the URL by looking for whitespace or common delimiters
	end := httpIndex + 7 // Start after "http://"
	if httpIndex <= len(s)-8 && s[httpIndex:httpIndex+8] == "https://" {
		end = httpIndex + 8 // Start after "https://"
	}

	// Continue until we hit whitespace, quotes, or other common delimiters
	for end < len(s) {
		char := s[end]
		if char == ' ' || char == '\t' || char == '\n' || char == '\r' ||
			char == '"' || char == '\'' || char == '`' ||
			char == '<' || char == '>' || char == '|' {
			break
		}
		end++
	}

	// Apply heuristics to trim common trailing punctuation
	for end > httpIndex+7 {
		lastChar := s[end-1]
		if lastChar == '.' || lastChar == ',' || lastChar == ';' || lastChar == ':' ||
			lastChar == '!' || lastChar == '?' || lastChar == ')' || lastChar == ']' ||
			lastChar == '}' || lastChar == '"' || lastChar == '\'' {
			end--
		} else {
			break
		}
	}

	// Make sure we have a reasonable URL length
	if end <= httpIndex+7 {
		return nil
	}

	return &Position{Start: httpIndex, End: end}
}

// processNode takes a node and splits it based on positions returned by findFn
// Non-found parts use the original node's type, found parts use the specified nodeType
func processNode(node *Node, findFn func(s string) *Position, nodeType string) []*Node {
	if node.Type != NodeTypeText {
		return []*Node{node}
	}

	var result []*Node
	content := node.Content
	offset := 0
	foundAny := false

	for {
		// Find the next position in the remaining content
		pos := findFn(content[offset:])
		if pos == nil {
			break
		}

		foundAny = true

		// Adjust position to absolute offset
		absoluteStart := offset + pos.Start
		absoluteEnd := offset + pos.End

		// Add text before this match (if any)
		if absoluteStart > offset {
			textContent := content[offset:absoluteStart]
			result = append(result, &Node{Type: node.Type, Content: textContent})
		}

		// Add the found content with the specified type
		foundContent := content[absoluteStart:absoluteEnd]
		result = append(result, &Node{Type: nodeType, Content: foundContent})

		// Move offset past this match
		offset = absoluteEnd
	}

	// If no matches were found, return original node
	if !foundAny {
		return []*Node{node}
	}

	// Add remaining text after last match (if any)
	if offset < len(content) {
		textContent := content[offset:]
		result = append(result, &Node{Type: node.Type, Content: textContent})
	}

	return result
}

// processNodeList applies processNode to every node in a linked list
// It takes the head of a linked list, a find function, and node type, then iterates through
// each node, applies processNode, and splices the results back into the list
func processNodeList(head *Node, findFn func(s string) *Position, nodeType string) *Node {
	if head == nil {
		return nil
	}

	current := head
	var newHead *Node

	for current != nil {
		// Process the current node using processNode
		processedNodes := processNode(current, findFn, nodeType)

		// Splice the processed nodes back into the list
		resultHead, nextNode := current.Splice(processedNodes)

		// Update the head if this was the first node
		if newHead == nil {
			newHead = resultHead
		}

		// Move to the next node (which is now the node after our spliced content)
		current = nextNode
	}

	return newHead
}

func ProcessLine(line string) *Node {
	head := &Node{Type: NodeTypeText, Content: line}
	head = processNodeList(head, FindFirstAnsiEscape, NodeTypeAnsi)
	head = processNodeList(head, func(s string) *Position {
		return FindFirstJSON(s, false) // Don't allow arrays
	}, NodeTypeJSON)
	head = processNodeList(head, FindFirstUUID, NodeTypeUUID)
	head = processNodeList(head, FindFirstURL, NodeTypeURL)
	return head
}
