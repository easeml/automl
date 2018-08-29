package dataset

import (
	"bufio"
	"fmt"
	"path"
	"strconv"
	"strings"
)

// InstanceID is.
type InstanceID struct {
	Node  string
	Index int
}

// Link is.
type Link struct {
	Src InstanceID
	Dst InstanceID
}

// Links is.
type Links struct {
	Name  string
	Links map[Link]interface{}
}

// Type is.
func (f Links) Type() string { return "links" }

func loadInstanceID(input string) (*InstanceID, error) {
	splits := strings.Split(input, "/")
	if len(splits) > 2 {
		return nil, &datasetError{err: "Wrong instance ID format."}
	} else if len(splits) == 2 {
		index, err := strconv.Atoi(splits[1])
		if err != nil {
			return nil, err
		}
		return &InstanceID{Node: splits[0], Index: index}, nil
	} else {
		return &InstanceID{Node: splits[0], Index: -1}, nil
	}
}

func (id *InstanceID) dump() string {
	if id.Index >= 0 {
		return fmt.Sprintf("%s/%d", id.Node, id.Index)
	}
	return fmt.Sprintf("%s", id.Node)
}

func loadLink(input string) (*Link, error) {
	fields := strings.Fields(input)
	if len(fields) != 2 {
		return nil, &datasetError{err: "Wrong link format."}
	}
	var err error
	var src, dst *InstanceID
	src, err = loadInstanceID(fields[0])
	if err != nil {
		return nil, err
	}
	dst, err = loadInstanceID(fields[1])
	if err != nil {
		return nil, err
	}

	return &Link{Src: *src, Dst: *dst}, nil
}

func (l *Link) dump() string {
	return fmt.Sprintf("%s %s", l.Src.dump(), l.Dst.dump())
}

func (l *Link) getReverse() *Link {
	return &Link{Src: l.Dst, Dst: l.Src}
}

func loadLinks(root string, relPath string, name string, opener Opener, metadataOnly bool) (*Links, error) {
	path := path.Join(relPath, name+TypeExtensions["links"])
	file, err := opener.GetFile(root, path, true, false)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	links := map[Link]interface{}{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var err error
		var link *Link
		line := scanner.Text()
		link, err = loadLink(strings.TrimSpace(line))
		if err != nil {
			return nil, err
		}
		links[*link] = nil
	}

	return &Links{Name: name, Links: links}, nil
}

func (f *Links) dump(root string, relPath string, name string, opener Opener) error {
	path := path.Join(relPath, name) + TypeExtensions["links"]
	file, err := opener.GetFile(root, path, false, false)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for l := range f.Links {
		fmt.Fprintln(writer, l.dump())
	}
	return writer.Flush()
}

// IsUndirected is.
func (f *Links) IsUndirected() bool {
	for l := range f.Links {
		// If for every link from A to B, we don't find
		// a link from B to A, then the graph is not undirected.
		if _, ok := f.Links[*l.getReverse()]; ok == false {
			return false
		}
	}
	return true
}

// IsFanin is.
func (f *Links) IsFanin(undirected bool) bool {
	dstNodes := map[InstanceID]int{}
	for l := range f.Links {
		// If a node has more than one incoming link, this constitutes a fan-in in a directed graph.
		// In an undirected graph, a fan-in happens when a node is connected to more than 2 nodes.
		count := dstNodes[l.Dst]
		if (undirected && count > 2) || (!undirected && count > 1) {
			return false
		}
		dstNodes[l.Dst] = count + 1
	}
	return false
}

// IsCyclic is.
func (f *Links) IsCyclic(undirected bool) bool {

	// This is a set of unvisited nodes.
	nodes := map[InstanceID]interface{}{}

	// Build adjacency list for each node.
	adjacency := map[InstanceID][]InstanceID{}
	for l := range f.Links {
		nodes[l.Src] = nil
		nodes[l.Dst] = nil
		if adjList, ok := adjacency[l.Src]; ok {
			adjacency[l.Src] = append(adjList, l.Dst)
		} else {
			adjacency[l.Src] = []InstanceID{l.Dst}
		}
	}

	// The algorithm differs for undirected and directed graphs.
	if undirected {
		for len(nodes) > 0 {

			// Get arbitrary unvisited node.
			var x InstanceID
			for i := range nodes {
				x = i
				delete(nodes, x)
				break
			}

			// For undirected graphs we need to remember the parent x.
			type elem struct {
				node   InstanceID
				parent InstanceID
			}
			stack := []elem{}
			for _, y := range adjacency[x] {
				stack = append(stack, elem{node: y, parent: x})
			}

			for len(stack) > 0 {

				// Get the node and its parent.
				var x elem
				x, stack = stack[len(stack)-1], stack[:len(stack)-1]

				// A node is missing from nodes only if it is visited.
				if _, ok := nodes[x.node]; ok == false {
					return true
				}
				delete(nodes, x.node)

				// In undirected graphs, all edges are bidirectional. We don't count this as a cycle.
				for _, y := range adjacency[x.node] {
					if y != x.parent {
						stack = append(stack, elem{node: y, parent: x.node})
					}
				}
			}
		}
	} else {
		for len(nodes) > 0 {

			// We keep a set of nodes that are ancestors in the DFS tree.
			ancestors := map[InstanceID]interface{}{}

			// Get arbitrary unvisited node and push it to the stack.
			var x InstanceID
			for i := range nodes {
				x = i
				break
			}
			stack := []InstanceID{x}

			for len(stack) > 0 {

				// We encounter each node twice.
				x := stack[len(stack)-1]

				// If it is not in ancestors, then this is a first encounter.
				if _, ok := ancestors[x]; ok == false {

					// Mark node as visited and add it to active ancestors.
					delete(nodes, x)
					ancestors[x] = nil

					// A cycle is detected if we find a back edge (i.e. edge pointing to an ancestor).
					for _, y := range adjacency[x] {
						if _, ok := ancestors[y]; ok {
							return true
						}
					}

				} else {
					// Since we encountered x the second time, we can pop it
					// from the stack and remove it from active ancestors.
					stack = stack[:len(stack)-1]
					delete(ancestors, x)
				}
			}
		}
	}

	// No cycle was found.
	return false
}

type destCountKey struct {
	src     InstanceID
	dstNode string
}

func (f *Links) getLinkDestinationCounts() map[destCountKey]int {
	result := map[destCountKey]int{}
	for l := range f.Links {
		key := destCountKey{src: l.Src, dstNode: l.Dst.Node}
		result[key] = result[key] + 1
	}
	return result
}

func (f *Links) getMaxIndicesPerNode() map[string]int {
	result := map[string]int{}
	for l := range f.Links {
		var index int
		index = result[l.Src.Node]
		if l.Src.Index > index {
			result[l.Src.Node] = l.Src.Index
		}
		index = result[l.Dst.Node]
		if l.Dst.Index > index {
			result[l.Src.Node] = l.Dst.Index
		}
	}
	return result
}
