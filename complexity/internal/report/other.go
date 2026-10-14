package report

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"strings"

	"github.com/ryangerardwilson/c3po/complexity/internal/metric"
)

func writeCSV(w io.Writer, fns []metric.Function, hotspotsOnly bool) error {
	fmt.Fprintln(w, "file,line,end_line,name,language,cognitive,nesting,lines,params,cyclomatic,violations")
	for _, fn := range fns {
		if hotspotsOnly && len(fn.Violations) == 0 {
			continue
		}
		fmt.Fprintf(w, "%s,%d,%d,%s,%s,%d,%d,%d,%d,%d,%s\n",
			csv(fn.File), fn.Line, fn.EndLine, csv(fn.Name), fn.Language,
			fn.Cognitive, fn.Nesting, fn.Lines, fn.Params, fn.Cyclomatic, csv(strings.Join(fn.Violations, "|")))
	}
	return nil
}

func csv(s string) string {
	if strings.ContainsAny(s, ",\"\n") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}

type sarifLog struct {
	Version string     `json:"version"`
	Schema  string     `json:"$schema"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name  string      `json:"name"`
	Rules []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type sarifResult struct {
	RuleID    string       `json:"ruleId"`
	Level     string       `json:"level"`
	Message   sarifMessage `json:"message"`
	Locations []sarifLoc   `json:"locations"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifLoc struct {
	Physical sarifPhys `json:"physicalLocation"`
}

type sarifPhys struct {
	Artifact sarifArt `json:"artifactLocation"`
	Region   sarifReg `json:"region"`
}

type sarifArt struct {
	URI string `json:"uri"`
}

type sarifReg struct {
	StartLine int `json:"startLine"`
}

func writeSARIF(w io.Writer, doc Document) error {
	rules := []string{"cognitive", "nesting", "lines", "params", "cyclomatic", "degree", "depth", "m2m", "cycles", "naming", "column_order"}
	log := sarifLog{
		Version: "2.1.0",
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Runs:    []sarifRun{{Tool: sarifTool{Driver: sarifDriver{Name: "complexity"}}}},
	}
	for _, id := range rules {
		log.Runs[0].Tool.Driver.Rules = append(log.Runs[0].Tool.Driver.Rules, sarifRule{ID: "complexity/" + id, Name: id})
	}
	for _, fn := range doc.Hotspots {
		for _, v := range fn.Violations {
			log.Runs[0].Results = append(log.Runs[0].Results, result("complexity/"+v, fn.File, fn.Line,
				fmt.Sprintf("%s %s %s", fn.Name, v, metricText(fn, v))))
		}
	}
	if doc.Models != nil {
		for _, m := range doc.Models.Hotspots {
			for _, v := range m.Violations {
				log.Runs[0].Results = append(log.Runs[0].Results, result("complexity/"+v, m.File, m.Line,
					fmt.Sprintf("%s %s", m.Name, v)))
			}
		}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(log)
}

func result(rule, file string, line int, text string) sarifResult {
	if line < 1 {
		line = 1
	}
	return sarifResult{
		RuleID: rule, Level: "error", Message: sarifMessage{Text: text},
		Locations: []sarifLoc{{Physical: sarifPhys{Artifact: sarifArt{URI: file}, Region: sarifReg{StartLine: line}}}},
	}
}

func metricText(fn metric.Function, v string) string {
	switch v {
	case "cognitive":
		return fmt.Sprint(fn.Cognitive)
	case "nesting":
		return fmt.Sprint(fn.Nesting)
	case "lines":
		return fmt.Sprint(fn.Lines)
	case "params":
		return fmt.Sprint(fn.Params)
	default:
		return fmt.Sprint(fn.Cyclomatic)
	}
}

func writeHTML(w io.Writer, doc Document, fns []metric.Function) error {
	fmt.Fprintf(w, "<!DOCTYPE html><html><head><meta charset=\"utf-8\"><title>complexity</title><style>body{font-family:sans-serif}table{border-collapse:collapse}td,th{border:1px solid #ccc;padding:4px 8px}</style></head><body>\n")
	fmt.Fprintf(w, "<h1>complexity</h1><p>functions %d, hotspots %d, max cognitive %d</p>\n",
		doc.Summary.FunctionsAnalyzed, doc.Summary.Hotspots, doc.Summary.MaxCognitive)
	fmt.Fprintln(w, "<table><tr><th>file</th><th>name</th><th>cognitive</th><th>nesting</th><th>lines</th><th>params</th></tr>")
	for _, fn := range ranked(fns) {
		fmt.Fprintf(w, "<tr><td>%s:%d</td><td>%s</td><td>%d</td><td>%d</td><td>%d</td><td>%d</td></tr>\n",
			html.EscapeString(fn.File), fn.Line, html.EscapeString(fn.Name), fn.Cognitive, fn.Nesting, fn.Lines, fn.Params)
	}
	fmt.Fprintln(w, "</table></body></html>")
	return nil
}
