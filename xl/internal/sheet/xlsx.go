package sheet

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Excel stores a formula without the leading =. AVG is not an Excel function.
var avgCall = regexp.MustCompile(`(?i)\bAVG(\s*)\(`)

func isXLSX(data []byte) bool {
	return len(data) >= 4 && data[0] == 'P' && data[1] == 'K' && data[2] == 3 && data[3] == 4
}

func xlsxPath(path string) string {
	ext := filepathExt(path)
	if strings.EqualFold(ext, ".xlsx") {
		return path
	}
	if ext == "" {
		return path + ".xlsx"
	}
	return strings.TrimSuffix(path, ext) + ".xlsx"
}

func filepathExt(path string) string {
	i := strings.LastIndex(path, "/")
	base := path[i+1:]
	dot := strings.LastIndex(base, ".")
	if dot <= 0 {
		return ""
	}
	return base[dot:]
}

func excelFormula(src string) string {
	f := strings.TrimSpace(src)
	if strings.HasPrefix(f, "=") {
		f = strings.TrimSpace(f[1:])
	}
	return avgCall.ReplaceAllString(f, "AVERAGE$1(")
}

func (s *Sheet) MarshalXLSX() ([]byte, error) {
	sheets := []*Sheet{s}
	if s.book != nil && len(s.book.Sheets) > 0 {
		sheets = s.book.Sheets
	}
	parts := []struct{ name, body string }{
		{"[Content_Types].xml", contentTypes(len(sheets))},
		{"_rels/.rels", rootRelsXML},
		{"xl/workbook.xml", workbookXMLFor(sheets)},
		{"xl/_rels/workbook.xml.rels", workbookRels(len(sheets))},
		{"xl/styles.xml", stylesXML},
	}
	for i, sh := range sheets {
		parts = append(parts, struct{ name, body string }{
			fmt.Sprintf("xl/worksheets/sheet%d.xml", i+1),
			worksheetXML(sh),
		})
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, part := range parts {
		w, err := zw.Create(part.name)
		if err != nil {
			return nil, err
		}
		if _, err := io.WriteString(w, part.body); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func worksheetXML(s *Sheet) string {
	type cell struct {
		c, r int
		src  string
	}
	var cells []cell
	minC, minR := MaxCol, MaxRow
	maxC, maxR := -1, -1
	for a, src := range s.cells {
		if strings.TrimSpace(src) == "" {
			continue
		}
		cells = append(cells, cell{c: a.C, r: a.R, src: src})
		if a.C < minC {
			minC = a.C
		}
		if a.R < minR {
			minR = a.R
		}
		if a.C > maxC {
			maxC = a.C
		}
		if a.R > maxR {
			maxR = a.R
		}
	}
	sort.Slice(cells, func(i, j int) bool {
		if cells[i].r != cells[j].r {
			return cells[i].r < cells[j].r
		}
		return cells[i].c < cells[j].c
	})

	var sheet strings.Builder
	sheet.WriteString(xmlHeader)
	sheet.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`)
	if len(cells) > 0 {
		sheet.WriteString(`<dimension ref="`)
		sheet.WriteString(AddrLabel(minC, minR))
		if minC != maxC || minR != maxR {
			sheet.WriteByte(':')
			sheet.WriteString(AddrLabel(maxC, maxR))
		}
		sheet.WriteString(`"/>`)
	}
	sheet.WriteString(`<sheetData>`)
	for i := 0; i < len(cells); {
		row := cells[i].r
		sheet.WriteString(`<row r="`)
		sheet.WriteString(strconv.Itoa(row + 1))
		sheet.WriteString(`">`)
		for i < len(cells) && cells[i].r == row {
			writeCell(&sheet, cells[i].c, cells[i].r, cells[i].src, s.Value(cells[i].c, cells[i].r))
			i++
		}
		sheet.WriteString(`</row>`)
	}
	sheet.WriteString(`</sheetData></worksheet>`)

	return sheet.String()
}

func writeCell(b *strings.Builder, c, r int, src string, v Value) {
	ref := AddrLabel(c, r)
	b.WriteString(`<c r="`)
	b.WriteString(ref)
	if strings.HasPrefix(strings.TrimSpace(src), "=") {
		b.WriteString(`"><f>`)
		b.WriteString(xmlEscape(excelFormula(src)))
		b.WriteString(`</f>`)
		if v.Kind == Number {
			b.WriteString(`<v>`)
			b.WriteString(strconv.FormatFloat(v.Num, 'f', -1, 64))
			b.WriteString(`</v>`)
		} else if strings.HasPrefix(strings.TrimSpace(src), "=") && v.Kind == Empty {
			b.WriteString(`<v>0</v>`)
		}
		b.WriteString(`</c>`)
		return
	}
	if n, ok := parseLiteralNumber(src); ok {
		b.WriteString(`"><v>`)
		b.WriteString(strconv.FormatFloat(n, 'f', -1, 64))
		b.WriteString(`</v></c>`)
		return
	}
	text := xmlText(src)
	b.WriteString(`" t="inlineStr"><is><t`)
	if text != strings.TrimSpace(text) || strings.ContainsAny(text, "\n\r\t") {
		b.WriteString(` xml:space="preserve"`)
	}
	b.WriteString(`>`)
	b.WriteString(xmlEscape(text))
	b.WriteString(`</t></is></c>`)
}

func xmlEscape(s string) string {
	var buf bytes.Buffer
	_ = xml.EscapeText(&buf, []byte(s))
	return buf.String()
}

func xmlText(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == '\t' || r == '\n' || r == '\r' || r >= 0x20 {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func ParseXLSX(data []byte) (*Sheet, error) {
	s, err := parseXLSX(data)
	if err != nil {
		return nil, err
	}
	if s.book != nil {
		for _, sh := range s.book.Sheets {
			sh.ensureRoom()
			sh.Dirty = false
			sh.epoch = 0
		}
		s.book.Dirty = false
		s.book.epoch = 0
		return s, nil
	}
	s.ensureRoom()
	s.Dirty = false
	return s, nil
}

func parseXLSX(data []byte) (*Sheet, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("read xlsx: %w", err)
	}
	files := map[string]*zip.File{}
	for _, f := range zr.File {
		name := path.Clean(strings.ReplaceAll(f.Name, "\\", "/"))
		files[name] = f
	}
	wb, ok := files["xl/workbook.xml"]
	if !ok {
		return nil, errors.New("read xlsx: workbook")
	}
	wbXML, err := readZip(wb)
	if err != nil {
		return nil, fmt.Errorf("read xlsx: %w", err)
	}
	listed, err := workbookSheets(wbXML)
	if err != nil {
		return nil, err
	}
	if len(listed) > maxSheets {
		return nil, fmt.Errorf("xlsx has more than %d sheets", maxSheets)
	}
	rels, err := readOptional(files, "xl/_rels/workbook.xml.rels")
	if err != nil {
		return nil, err
	}
	targets := map[string]relTarget{}
	if rels != nil {
		targets, err = parseRels(rels)
		if err != nil {
			return nil, fmt.Errorf("read xlsx: %w", err)
		}
	}
	var sharedName string
	for _, target := range targets {
		if strings.HasSuffix(target.kind, "/sharedStrings") {
			sharedName = joinZip("xl/workbook.xml", target.target)
		}
	}
	var shared []string
	if sharedName != "" {
		raw, err := readOptional(files, sharedName)
		if err != nil {
			return nil, err
		}
		if raw != nil {
			shared, err = parseSharedStrings(raw)
			if err != nil {
				return nil, fmt.Errorf("read xlsx: %w", err)
			}
		}
	}
	if len(listed) == 0 {
		return Blank(), nil
	}
	book := &Book{}
	var first *Sheet
	for i, item := range listed {
		sheetName := "xl/worksheets/sheet1.xml"
		if item.id != "" {
			target, ok := targets[item.id]
			if !ok {
				return nil, fmt.Errorf("read xlsx: missing sheet %s", item.id)
			}
			sheetName = joinZip("xl/workbook.xml", target.target)
		}
		raw, ok := files[sheetName]
		if !ok {
			return nil, fmt.Errorf("read xlsx: missing %s", sheetName)
		}
		sheetXML, err := readZip(raw)
		if err != nil {
			return nil, fmt.Errorf("read xlsx: %w", err)
		}
		sh, err := readWorksheet(sheetXML, shared)
		if err != nil {
			return nil, err
		}
		sh.Name = uniqueSheetName(book, item.name)
		if sh.Name == "" {
			sh.Name = fmt.Sprintf("Sheet%d", i+1)
		}
		sh.adopt(book)
		if first == nil {
			first = sh
		}
	}
	return first, nil
}

type relTarget struct {
	kind   string
	target string
}

func readZip(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func readOptional(files map[string]*zip.File, name string) ([]byte, error) {
	f, ok := files[name]
	if !ok {
		return nil, nil
	}
	raw, err := readZip(f)
	if err != nil {
		return nil, fmt.Errorf("read xlsx: %w", err)
	}
	return raw, nil
}

func joinZip(baseFile, target string) string {
	target = strings.ReplaceAll(target, "\\", "/")
	if strings.HasPrefix(target, "/") {
		return strings.TrimPrefix(path.Clean(target), "/")
	}
	return path.Clean(path.Join(path.Dir(baseFile), target))
}

type workbookSheet struct {
	id   string
	name string
}

func workbookSheets(data []byte) ([]workbookSheet, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	var listed []workbookSheet
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return listed, nil
		}
		if err != nil {
			return nil, fmt.Errorf("read xlsx: %w", err)
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "sheet" {
			continue
		}
		item := workbookSheet{}
		for _, attr := range start.Attr {
			switch attr.Name.Local {
			case "id":
				item.id = attr.Value
			case "name":
				item.name = attr.Value
			}
		}
		if item.id == "" {
			return nil, errors.New("read xlsx: sheet has no id")
		}
		listed = append(listed, item)
	}
}

func parseRels(data []byte) (map[string]relTarget, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	out := map[string]relTarget{}
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "Relationship" {
			continue
		}
		var id, kind, target string
		for _, attr := range start.Attr {
			switch attr.Name.Local {
			case "Id":
				id = attr.Value
			case "Type":
				kind = attr.Value
			case "Target":
				target = attr.Value
			}
		}
		if id != "" && target != "" {
			out[id] = relTarget{kind: kind, target: target}
		}
	}
}

func parseSharedStrings(data []byte) ([]string, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	var out []string
	var cur strings.Builder
	inSI := false
	inT := false
	skip := 0
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		switch el := tok.(type) {
		case xml.StartElement:
			if el.Name.Local == "rPh" || el.Name.Local == "phoneticPr" {
				skip++
			}
			if el.Name.Local == "si" {
				inSI = true
				cur.Reset()
			}
			if inSI && skip == 0 && el.Name.Local == "t" {
				inT = true
			}
		case xml.EndElement:
			if el.Name.Local == "t" {
				inT = false
			}
			if el.Name.Local == "rPh" || el.Name.Local == "phoneticPr" {
				if skip > 0 {
					skip--
				}
			}
			if el.Name.Local == "si" {
				out = append(out, cur.String())
				inSI = false
			}
		case xml.CharData:
			if inT {
				cur.Write(el)
			}
		}
	}
}

type sharedFormula struct {
	src  string
	c, r int
}

type pendingFormula struct {
	c, r int
	si   string
}

func readWorksheet(data []byte, sharedStrings []string) (*Sheet, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	s := Blank()
	inSheet := false
	inCell := false
	inF := false
	inV := false
	inInline := false
	inT := false
	var ref, typ, formula, value string
	var inline strings.Builder
	var fKind, fSI string
	sawF := false
	shared := map[string]sharedFormula{}
	var pending []pendingFormula
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read xlsx: %w", err)
		}
		switch el := tok.(type) {
		case xml.StartElement:
			switch el.Name.Local {
			case "sheetData":
				inSheet = true
			case "c":
				if !inSheet {
					break
				}
				inCell = true
				ref, typ, formula, value = "", "", "", ""
				inline.Reset()
				sawF = false
				fKind, fSI = "", ""
				for _, attr := range el.Attr {
					switch attr.Name.Local {
					case "r":
						ref = attr.Value
					case "t":
						typ = attr.Value
					}
				}
			case "f":
				if inCell {
					inF = true
					sawF = true
					formula = ""
					fKind, fSI = "", ""
					for _, attr := range el.Attr {
						switch attr.Name.Local {
						case "t":
							fKind = attr.Value
						case "si":
							fSI = attr.Value
						}
					}
				}
			case "v":
				if inCell && !inF {
					inV = true
					value = ""
				}
			case "is":
				if inCell {
					inInline = true
				}
			case "t":
				if inInline {
					inT = true
				}
			}
		case xml.EndElement:
			switch el.Name.Local {
			case "sheetData":
				inSheet = false
			case "f":
				inF = false
			case "v":
				inV = false
			case "t":
				inT = false
			case "is":
				inInline = false
			case "c":
				if !inCell {
					break
				}
				inCell = false
				c, r, ok := parseCellRef(ref)
				if !ok {
					return nil, fmt.Errorf("read xlsx: cell %s", ref)
				}
				f := strings.TrimSpace(formula)
				if sawF && f == "" && fKind == "shared" && fSI != "" {
					if master, ok := shared[fSI]; ok {
						f = shiftFormula(master.src, c-master.c, r-master.r)
					} else {
						pending = append(pending, pendingFormula{c: c, r: r, si: fSI})
						break
					}
				}
				if sawF && f != "" && fKind == "shared" && fSI != "" {
					shared[fSI] = sharedFormula{src: f, c: c, r: r}
				}
				src, err := cellSource(sawF, f, typ, value, inline.String(), sharedStrings)
				if err != nil {
					return nil, err
				}
				if err := putCell(s, c, r, src); err != nil {
					return nil, err
				}
			}
		case xml.CharData:
			switch {
			case inF:
				formula += string(el)
			case inV:
				value += string(el)
			case inT:
				inline.Write(el)
			}
		}
	}
	for _, item := range pending {
		master, ok := shared[item.si]
		if !ok {
			return nil, fmt.Errorf("read xlsx: shared formula %s", item.si)
		}
		f := shiftFormula(master.src, item.c-master.c, item.r-master.r)
		src, err := cellSource(true, f, "", "", "", nil)
		if err != nil {
			return nil, err
		}
		if err := putCell(s, item.c, item.r, src); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func cellSource(sawF bool, formula, typ, value, inline string, sharedStrings []string) (string, error) {
	if sawF {
		f := strings.TrimSpace(formula)
		if f == "" {
			return "=", nil
		}
		if !strings.HasPrefix(f, "=") {
			f = "=" + f
		}
		return f, nil
	}
	switch typ {
	case "inlineStr":
		return inline, nil
	case "s":
		idx, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil || idx < 0 || idx >= len(sharedStrings) {
			return "", fmt.Errorf("read xlsx: shared string %s", strings.TrimSpace(value))
		}
		return sharedStrings[idx], nil
	case "b":
		if strings.TrimSpace(value) == "1" {
			return "TRUE", nil
		}
		if strings.TrimSpace(value) == "0" {
			return "FALSE", nil
		}
		return value, nil
	case "e":
		return value, nil
	default:
		value = strings.TrimSpace(value)
		if value == "" {
			return "", nil
		}
		n, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return value, nil
		}
		return formatNumber(n), nil
	}
}

func putCell(s *Sheet, c, r int, src string) error {
	if src == "" {
		return nil
	}
	if c >= MaxCol || r >= MaxRow || c < 0 || r < 0 {
		return fmt.Errorf("xlsx exceeds %d columns or %d rows", MaxCol, MaxRow)
	}
	s.cells[Addr{C: c, R: r}] = src
	return nil
}

func parseCellRef(ref string) (int, int, bool) {
	c, r, end, _, _, ok := cellRefAt(strings.TrimSpace(ref), 0)
	if !ok || end != len(strings.TrimSpace(ref)) {
		return 0, 0, false
	}
	return c, r, true
}

func cellRefAt(s string, i int) (c, r, end int, absC, absR, ok bool) {
	j := i
	if j < len(s) && s[j] == '$' {
		absC = true
		j++
	}
	nameStart := j
	for j < len(s) && isLetter(s[j]) {
		j++
	}
	nameEnd := j
	if nameEnd == nameStart || nameEnd-nameStart > 3 {
		return 0, 0, 0, false, false, false
	}
	if j < len(s) && s[j] == '$' {
		absR = true
		j++
	}
	rowStart := j
	for j < len(s) && isDigit(s[j]) {
		j++
	}
	if j == rowStart {
		return 0, 0, 0, false, false, false
	}
	col, good := ColIndex(s[nameStart:nameEnd])
	if !good {
		return 0, 0, 0, false, false, false
	}
	row, err := strconv.Atoi(s[rowStart:j])
	if err != nil || row < 1 {
		return 0, 0, 0, false, false, false
	}
	return col, row - 1, j, absC, absR, true
}

func shiftFormula(formula string, dc, dr int) string {
	if dc == 0 && dr == 0 {
		return formula
	}
	var b strings.Builder
	i := 0
	for i < len(formula) {
		if formula[i] == '"' {
			j := i + 1
			for j < len(formula) {
				if formula[j] != '"' {
					j++
					continue
				}
				j++
				if j < len(formula) && formula[j] == '"' {
					j++
					continue
				}
				break
			}
			b.WriteString(formula[i:j])
			i = j
			continue
		}
		if c, r, end, absC, absR, ok := cellRefAt(formula, i); ok {
			b.WriteString(shiftedRef(c, r, dc, dr, absC, absR))
			i = end
			continue
		}
		b.WriteByte(formula[i])
		i++
	}
	return b.String()
}

func shiftedRef(c, r, dc, dr int, absC, absR bool) string {
	if !absC {
		c += dc
	}
	if !absR {
		r += dr
	}
	if c < 0 || r < 0 {
		return "#REF!"
	}
	var b strings.Builder
	if absC {
		b.WriteByte('$')
	}
	b.WriteString(ColName(c))
	if absR {
		b.WriteByte('$')
	}
	b.WriteString(strconv.Itoa(r + 1))
	return b.String()
}

const xmlHeader = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n"

func contentTypes(n int) string {
	var b strings.Builder
	b.WriteString(xmlHeader)
	b.WriteString(`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">`)
	b.WriteString(`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>`)
	b.WriteString(`<Default Extension="xml" ContentType="application/xml"/>`)
	b.WriteString(`<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>`)
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, `<Override PartName="/xl/worksheets/sheet%d.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>`, i)
	}
	b.WriteString(`<Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>`)
	b.WriteString(`</Types>`)
	return b.String()
}

const rootRelsXML = xmlHeader + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
	`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>` +
	`</Relationships>`

func workbookXMLFor(sheets []*Sheet) string {
	var b strings.Builder
	b.WriteString(xmlHeader)
	b.WriteString(`<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets>`)
	for i, sh := range sheets {
		name := sh.Name
		if strings.TrimSpace(name) == "" {
			name = fmt.Sprintf("Sheet%d", i+1)
		}
		fmt.Fprintf(&b, `<sheet name="%s" sheetId="%d" r:id="rId%d"/>`, xmlAttr(name), i+1, i+1)
	}
	b.WriteString(`</sheets><calcPr fullCalcOnLoad="1"/></workbook>`)
	return b.String()
}

func xmlAttr(s string) string {
	return strings.ReplaceAll(xmlEscape(s), `"`, "&quot;")
}

func workbookRels(n int) string {
	var b strings.Builder
	b.WriteString(xmlHeader)
	b.WriteString(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet%d.xml"/>`, i, i)
	}
	fmt.Fprintf(&b, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>`, n+1)
	b.WriteString(`</Relationships>`)
	return b.String()
}

const stylesXML = xmlHeader + `<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">` +
	`<fonts count="1"><font><sz val="11"/><name val="Calibri"/></font></fonts>` +
	`<fills count="2"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill></fills>` +
	`<borders count="1"><border><left/><right/><top/><bottom/><diagonal/></border></borders>` +
	`<cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>` +
	`<cellXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/></cellXfs>` +
	`</styleSheet>`
