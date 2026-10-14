package glob

import "strings"

// Match reports whether path matches a slash-separated glob.
// "**" matches across directories. "*" and "?" match inside one segment.
func Match(pattern, path string) bool {
	pattern = strings.Trim(pattern, "/")
	path = strings.Trim(path, "/")
	if pattern == "" {
		return path == ""
	}
	return match(split(pattern), split(path))
}

func split(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, "/")
	out := parts[:0]
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func match(pat, segs []string) bool {
	ip, is := 0, 0
	starP, starS := -1, -1
	for is < len(segs) {
		if ip < len(pat) && pat[ip] == "**" {
			starP, starS = ip, is
			ip++
			continue
		}
		if ip < len(pat) && seg(pat[ip], segs[is]) {
			ip++
			is++
			continue
		}
		if starP < 0 {
			return false
		}
		starS++
		is = starS
		ip = starP + 1
	}
	for ip < len(pat) && pat[ip] == "**" {
		ip++
	}
	return ip == len(pat)
}

func seg(pat, s string) bool {
	i, j := 0, 0
	star, mark := -1, 0
	for j < len(s) {
		if i < len(pat) && (pat[i] == '?' || pat[i] == s[j]) {
			i++
			j++
			continue
		}
		if i < len(pat) && pat[i] == '*' {
			star, mark = i, j
			i++
			continue
		}
		if star < 0 {
			return false
		}
		mark++
		i, j = star+1, mark
	}
	for i < len(pat) && pat[i] == '*' {
		i++
	}
	return i == len(pat)
}
