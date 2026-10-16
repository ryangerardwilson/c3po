package vibetree

import (
	"encoding/json"
	"io"
)

type jsonReport struct {
	Mode    string    `json:"mode"`
	Modules int       `json:"modules"`
	P       int       `json:"p"`
	Cutoff  int       `json:"cutoff"`
	Note    string    `json:"note,omitempty"`
	Tree    *jsonNode `json:"tree"`
}

type jsonNode struct {
	Path     string     `json:"path"`
	Name     string     `json:"name"`
	Size     int        `json:"size"`
	Children []jsonNode `json:"children,omitempty"`
}

// writeJSONReport prints the same cutoff as the text report, as one JSON object.
func writeJSONReport(w io.Writer, root *node, p int, mode string) {
	all := sizes(root)
	rep := jsonReport{Mode: mode, Modules: len(all), P: p}
	if len(all) > 0 {
		rep.Cutoff = percentile(all, p)
		if root.size >= rep.Cutoff {
			node := agentNode(root, "", rep.Cutoff)
			rep.Tree = &node
		}
		rep.Note = cutoffNote(len(all), p, root, rep.Cutoff)
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(rep)
}

func agentNode(n *node, parent string, cut int) jsonNode {
	path := n.name
	if parent != "" && parent != "." {
		path = parent + "/" + n.name
	}
	out := jsonNode{Path: path, Name: n.name, Size: n.size}
	for _, kid := range n.kids {
		if kid.size < cut {
			continue
		}
		out.Children = append(out.Children, agentNode(kid, path, cut))
	}
	return out
}
