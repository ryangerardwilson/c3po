package models

func annotate(models []Model) {
	incoming := map[string]int{}
	adj := map[string][]string{}
	for i := range models {
		m := &models[i]
		m.FanOut = len(m.Relationships)
		for _, r := range m.Relationships {
			incoming[r.Target]++
			if r.Type == "ManyToMany" {
				m.M2M++
			}
			if r.Type == "ForeignKey" || r.Type == "OneToOne" {
				adj[m.Name] = append(adj[m.Name], r.Target)
			}
		}
	}
	for i := range models {
		m := &models[i]
		m.FanIn = incoming[m.Name]
		m.Degree = m.FanIn + m.FanOut
		m.Depth = longest(adj, m.Name)
		m.Cycles = cycled(adj, m.Name)
	}
}

func longest(adj map[string][]string, start string) int {
	seen := map[string]bool{}
	return walkLong(adj, start, seen)
}

func walkLong(adj map[string][]string, node string, seen map[string]bool) int {
	seen[node] = true
	best := 0
	for _, next := range adj[node] {
		if seen[next] {
			continue
		}
		d := 1 + walkLong(adj, next, seen)
		if d > best {
			best = d
		}
	}
	delete(seen, node)
	return best
}

func cycled(adj map[string][]string, start string) bool {
	color := map[string]int{}
	return cycleFrom(adj, start, color)
}

func cycleFrom(adj map[string][]string, node string, color map[string]int) bool {
	color[node] = 1
	for _, next := range adj[node] {
		if color[next] == 1 || (color[next] == 0 && cycleFrom(adj, next, color)) {
			return true
		}
	}
	color[node] = 2
	return false
}
