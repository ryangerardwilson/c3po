package gates

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/ryangerardwilson/c3po/internal/version"
)

// receipt is the first 8 hex characters of SHA-256 over the c3po version,
// the canonical section, and the body, joined by newlines. It shows that a
// body was fetched from this build.
func receipt(p provision) string {
	sum := sha256.Sum256([]byte(version.String() + "\n" + p.section + "\n" + p.body))
	return hex.EncodeToString(sum[:])[:8]
}

// printSections prints the bodies of the sections named, in the order
// written. One section is its body and receipt. Several are each labelled
// with the published section and separated by a blank line. A section with
// subsections lists them under its body. An unknown section prints
// nothing at all.
func printSections(args []string, k kind, s statute, stdout, stderr io.Writer) int {
	ps := make([]provision, 0, len(args))
	shown := map[string]bool{}
	for _, arg := range args {
		p, ok := s.find(arg)
		if !ok {
			return usageError(stderr, "unknown section %q in %s %s", arg, k.name, s.name)
		}
		ps = append(ps, p)
		shown[p.section] = true
	}
	for i, p := range ps {
		if i > 0 {
			fmt.Fprintln(stdout)
		}
		if len(ps) > 1 {
			fmt.Fprintln(stdout, p.section)
		}
		printBody(stdout, k, s, p, shown)
	}
	return 0
}

// printBody prints a body and its receipt and, when the section has
// subsections, a blank line, one heading line per immediate subsection,
// and a closing line for the subsection bodies not in shown.
func printBody(stdout io.Writer, k kind, s statute, p provision, shown map[string]bool) {
	fmt.Fprintln(stdout, p.body)
	fmt.Fprintln(stdout, "receipt "+receipt(p))
	subs := s.subsections(p.section)
	if len(subs) == 0 {
		return
	}
	fmt.Fprintln(stdout)
	printHeadings(stdout, subs)
	var missing []provision
	for _, sub := range subs {
		if !shown[sub.section] {
			missing = append(missing, sub)
		}
	}
	if len(missing) > 0 {
		fmt.Fprintf(stdout, "%s not shown: c3po gates %s %s %s\n",
			bodies(len(missing)), k.name, s.name, fetchArgs(p.section, missing))
	}
}

// printAll prints every provision in section order as its section, body,
// and receipt, with a blank line between them and no subsection list.
func printAll(stdout io.Writer, s statute) {
	for i, p := range s.provisions {
		if i > 0 {
			fmt.Fprintln(stdout)
		}
		fmt.Fprintln(stdout, p.section)
		fmt.Fprintln(stdout, p.body)
		fmt.Fprintln(stdout, "receipt "+receipt(p))
	}
}

func bodies(n int) string {
	if n == 1 {
		return "1 body"
	}
	return strconv.Itoa(n) + " bodies"
}

// fetchArgs writes the missing subsections of parent as section arguments
// without the section sign: parent.{1..N} when they are exactly the first
// N subsections, otherwise each section separated by a space.
func fetchArgs(parent string, missing []provision) string {
	run := len(missing) > 1
	for i, p := range missing {
		run = run && p.section == parent+"."+strconv.Itoa(i+1)
	}
	if run {
		return strings.TrimPrefix(parent, "§") + ".{1.." + strconv.Itoa(len(missing)) + "}"
	}
	return bareSections(missing)
}

// bareSections joins sections without the section sign, ready to follow
// the statute as arguments.
func bareSections(ps []provision) string {
	words := make([]string, len(ps))
	for i, p := range ps {
		words[i] = strings.TrimPrefix(p.section, "§")
	}
	return strings.Join(words, " ")
}
