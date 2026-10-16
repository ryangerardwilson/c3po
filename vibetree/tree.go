package vibetree

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

type node struct {
	name string
	size int
	kids []*node
	by   map[string]*node
}

type view struct {
	cut   int
	max   int
	floor int
	color bool
}

func build(files []item) *node {
	root := &node{name: ".", by: map[string]*node{}}
	for _, f := range files {
		place(root, f)
	}
	total(root)
	sortTree(root)
	return root
}

func place(root *node, f item) {
	cur := root
	parts := strings.Split(f.path, "/")
	for i, part := range parts {
		if part == "" || part == "." {
			continue
		}
		child := cur.by[part]
		if child == nil {
			child = &node{name: part, by: map[string]*node{}}
			cur.by[part] = child
			cur.kids = append(cur.kids, child)
		}
		cur = child
		if i == len(parts)-1 {
			cur.size = f.size
		}
	}
}

func total(n *node) int {
	if len(n.kids) == 0 {
		return n.size
	}
	sum := 0
	for _, k := range n.kids {
		sum += total(k)
	}
	n.size = sum
	return sum
}

func sortTree(n *node) {
	sort.Slice(n.kids, func(i, j int) bool {
		if n.kids[i].size != n.kids[j].size {
			return n.kids[i].size > n.kids[j].size
		}
		return n.kids[i].name < n.kids[j].name
	})
	for _, k := range n.kids {
		sortTree(k)
	}
}

func sizes(n *node) []int {
	var out []int
	each(n, func(m *node) {
		if m.size > 0 {
			out = append(out, m.size)
		}
	})
	sort.Ints(out)
	return out
}

func each(n *node, fn func(*node)) {
	fn(n)
	for _, k := range n.kids {
		each(k, fn)
	}
}

// percentile is the nearest rank of p, with p from 1 to 99, over ascending values.
func percentile(sorted []int, p int) int {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	return sorted[nearestRank(n, p)-1]
}

func nearestRank(n, p int) int {
	if n == 0 {
		return 0
	}
	rank := (p*n + 99) / 100
	if rank < 1 {
		return 1
	}
	if rank > n {
		return n
	}
	return rank
}

func writeReport(w io.Writer, root *node, p int, mode string, color bool) {
	all := sizes(root)
	fmt.Fprintf(w, "mode %s\nmodules %d\n", mode, len(all))
	if len(all) == 0 {
		fmt.Fprintln(w, "No modules.")
		return
	}
	cut := percentile(all, p)
	fmt.Fprintf(w, "p%d %d\n", p, cut)
	writeNode(w, root, view{cut: cut, max: root.size, floor: floorOf(mode), color: color}, 0)
	if note := cutoffNote(len(all), p, root, cut); note != "" {
		fmt.Fprintf(w, "\n%s\n", note)
	}
}

func cutoffNote(n, p int, root *node, cut int) string {
	rank := nearestRank(n, p)
	if rank < n || !soleRoot(root, cut) {
		return ""
	}
	return fmt.Sprintf("p%d of %d modules is rank %d, the largest module. The root is that size because a directory sums its files. Try a lower -p.", p, n, rank)
}

func soleRoot(root *node, cut int) bool {
	if root.size < cut {
		return false
	}
	for _, k := range root.kids {
		if k.size >= cut {
			return false
		}
	}
	return true
}

func writeNode(w io.Writer, n *node, v view, depth int) {
	if n.size < v.cut {
		return
	}
	line := fmt.Sprintf("%s%s  %d", strings.Repeat("  ", depth), n.name, n.size)
	fmt.Fprintln(w, paint(line, n.size, v.max, v.floor, v.color))
	for _, k := range n.kids {
		writeNode(w, k, v, depth+1)
	}
}
