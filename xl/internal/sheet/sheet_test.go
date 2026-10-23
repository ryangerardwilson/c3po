package sheet

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMonthlyBudget(t *testing.T) {
	s := Blank()
	set := func(addr, src string) {
		t.Helper()
		c, r, ok := mustAddr(t, addr)
		if !ok {
			t.Fatalf("bad addr %s", addr)
		}
		s.Set(c, r, src)
	}
	set("A1", "Expenses")
	set("A2", "Rent")
	set("B2", "1200")
	set("A3", "Food")
	set("B3", "450")
	set("A4", "Transit")
	set("B4", "120")
	set("A5", "Total")
	set("B5", "=SUM(B2:B4)")
	set("D1", "Income")
	set("D2", "Pay")
	set("E2", "4000")
	set("D3", "Other")
	set("E3", "150")
	set("D4", "Total")
	set("E4", "=SUM(E2:E3)")
	set("A7", "Savings")
	set("B7", "=E4-B5")

	if got := s.Display(1, 4); got != "1770" {
		t.Fatalf("expenses = %s", got)
	}
	if got := s.Display(4, 3); got != "4150" {
		t.Fatalf("income = %s", got)
	}
	if got := s.Display(1, 6); got != "2380" {
		t.Fatalf("savings = %s", got)
	}

	stats := s.Stats(1, 1, 1, 3)
	if stats.String() != "Sum 1,770   Average 590   Median 450   Mode none   n 3" {
		t.Fatalf("column stats %s", stats.String())
	}
	row := s.Stats(4, 1, 4, 2)
	if row.Sum != 4150 || row.Count != 2 {
		t.Fatalf("income row %+v", row)
	}
	if !s.PureNumbers(1, 1, 1, 3) || s.PureNumbers(0, 0, 1, 3) {
		t.Fatal("pure numbers")
	}
}

func TestFormulaLanguage(t *testing.T) {
	s := Blank()
	s.Set(0, 0, "hello")
	s.Set(0, 1, "5")
	s.Set(0, 2, "7")
	s.Set(1, 0, "1,200")
	s.Set(1, 1, "300")
	s.Set(2, 0, "12,00")
	cases := []struct {
		cell string
		src  string
		want string
	}{
		{"C3", "=2+3*4", "14"},
		{"C4", "=(2+3)*4", "20"},
		{"C5", "=1/0", "#DIV/0"},
		{"C6", "=SUM(A1:A3)", "12"},
		{"C7", "=AVERAGE(A2:A3)", "6"},
		{"C8", "=MEDIAN(A2:A3)", "6"},
		{"C9", "=COUNT(A1:A3)", "2"},
		{"C10", "=SUM(B1:B2)", "1500"},
		{"C11", "=SUM(H1)", "0"},
		{"C16", "=MEDIAN(B1:B2)", "750"},
		{"C12", "=NOSUCH(1)", "#NAME"},
		{"C13", "=SUM(", "#ERR"},
		{"C14", "=-B2", "-300"},
		{"C15", "=SUM(B1:B2,A2)", "1505"},
		{"D1", "=A1", "hello"},
		{"D2", "=Z99", "0"},
	}
	for _, tc := range cases {
		c, r, _ := mustAddr(t, tc.cell)
		s.Set(c, r, tc.src)
		if got := s.Display(c, r); got != tc.want {
			t.Fatalf("%s %s -> %s, want %s", tc.cell, tc.src, got, tc.want)
		}
	}

	s.Set(3, 2, "10")
	s.Set(3, 3, "10")
	s.Set(3, 4, "20")
	s.Set(3, 5, "20")
	s.Set(4, 2, "=MODE(D3:D6)")
	if got := s.Display(4, 2); got != "10" {
		t.Fatalf("mode tie = %s", got)
	}
	s.Set(4, 3, "=MODE(A2:A3)")
	if got := s.Display(4, 3); got != "#N/A" {
		t.Fatalf("mode unique = %s", got)
	}
	s.Set(4, 4, "=MEDIAN(D3:D5)")
	if got := s.Display(4, 4); got != "10" {
		t.Fatalf("median odd = %s", got)
	}

	s.Set(5, 0, "=F2")
	s.Set(5, 1, "=F1")
	if got := s.Display(5, 0); got != "#CYCLE" {
		t.Fatalf("cycle = %s", got)
	}
	if s.Display(2, 0) != "12,00" {
		t.Fatalf("bad thousands stayed text, got %s", s.Display(2, 0))
	}
}

func TestXLSXRoundTripKeepsFormulas(t *testing.T) {
	s := Blank()
	s.Set(0, 0, "Rent, utilities")
	s.Set(0, 1, "A & B")
	s.Set(1, 0, "=SUM(B2:B3)")
	s.Set(1, 1, "10")
	s.Set(1, 2, "15")
	s.Set(2, 0, "=AVG(B2)")
	s.Set(3, 0, "1,200")
	dir := t.TempDir()
	s.Path = filepath.Join(dir, "budget.csv")
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	if s.Dirty {
		t.Fatal("save left the sheet dirty")
	}
	if filepath.Base(s.Path) != "budget.xlsx" {
		t.Fatalf("path %s", s.Path)
	}
	if _, err := os.Stat(filepath.Join(dir, "budget.csv")); !os.IsNotExist(err) {
		t.Fatal("save wrote a csv")
	}
	raw, err := os.ReadFile(s.Path)
	if err != nil {
		t.Fatal(err)
	}
	xml := zipSheetXML(t, raw)
	if !strings.Contains(xml, "<f>SUM(B2:B3)</f>") || !strings.Contains(xml, "<f>AVERAGE(B2)</f>") {
		t.Fatalf("formula was not stored:\n%s", xml)
	}
	loaded, err := Open(s.Path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Display(1, 0) != "25" || loaded.Display(2, 0) != "10" {
		t.Fatalf("sum %s avg %s", loaded.Display(1, 0), loaded.Display(2, 0))
	}
	if loaded.Source(0, 0) != "Rent, utilities" || loaded.Source(0, 1) != "A & B" {
		t.Fatalf("label %q amp %q", loaded.Source(0, 0), loaded.Source(0, 1))
	}
	if loaded.Source(2, 0) != "=AVERAGE(B2)" || loaded.Source(3, 0) != "1200" {
		t.Fatalf("avg %q thousands %q", loaded.Source(2, 0), loaded.Source(3, 0))
	}
	if loaded.Dirty {
		t.Fatal("loaded sheet is dirty")
	}
}

func TestCSVImportSavesXLSX(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "old.csv")
	body := "Rent,=SUM(B2:B3)\n,10\n,15\n"
	if err := os.WriteFile(csvPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Open(csvPath)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(s.Path) != "old.xlsx" {
		t.Fatal(s.Path)
	}
	if _, err := os.Stat(filepath.Join(dir, "old.xlsx")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(csvPath); err != nil {
		t.Fatal("csv removed")
	}
	if s.Source(0, 0) != "Rent" || s.Display(1, 0) != "25" {
		t.Fatalf("import %q %s", s.Source(0, 0), s.Display(1, 0))
	}
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	if filepath.Base(s.Path) != "old.xlsx" {
		t.Fatal(s.Path)
	}
	loaded, err := Open(s.Path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Source(1, 0) != "=SUM(B2:B3)" || loaded.Display(1, 0) != "25" {
		t.Fatalf("source %q display %s", loaded.Source(1, 0), loaded.Display(1, 0))
	}
}

func TestXLSXReadsSpreadsheetFormula(t *testing.T) {
	sheetXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>
<row r="1">
<c r="A1" t="s"><v>0</v></c>
<c r="B1"><v>10</v></c>
<c r="C1"><f t="shared" ref="C1:C2" si="0">SUM(B1:B2)</f><v>25</v></c>
</row>
<row r="2">
<c r="B2"><v>15</v></c>
<c r="C2"><f t="shared" si="0"/></c>
</row>
</sheetData></worksheet>`
	sst := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><si><t>Rent &amp; food</t></si></sst>`
	wb := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Sheet1" sheetId="1" r:id="rId1"/></sheets></workbook>`
	rels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/sharedStrings" Target="sharedStrings.xml"/></Relationships>`
	data := zipParts(t, map[string]string{
		"xl/workbook.xml":            wb,
		"xl/_rels/workbook.xml.rels": rels,
		"xl/sharedStrings.xml":       sst,
		"xl/worksheets/sheet1.xml":   sheetXML,
	})
	s, err := ParseXLSX(data)
	if err != nil {
		t.Fatal(err)
	}
	if s.Source(0, 0) != "Rent & food" {
		t.Fatalf("label %q", s.Source(0, 0))
	}
	if s.Source(2, 0) != "=SUM(B1:B2)" || s.Display(2, 0) != "25" {
		t.Fatalf("C1 %q %s", s.Source(2, 0), s.Display(2, 0))
	}
	if s.Source(2, 1) != "=SUM(B2:B3)" {
		t.Fatalf("shifted %q", s.Source(2, 1))
	}
}

func TestCrossSheetFormula(t *testing.T) {
	s := Blank()
	if s.Name != "Sheet1" || s.SheetCount() != 1 {
		t.Fatalf("blank %s count %d", s.Name, s.SheetCount())
	}
	income, err := s.AddSheet("Income")
	if err != nil {
		t.Fatal(err)
	}
	income.Set(1, 0, "4000")
	income.Set(1, 1, "150")
	s.Set(1, 0, "=SUM(Income!B1:B2)")
	if s.Display(1, 0) != "4150" {
		t.Fatalf("sum %s", s.Display(1, 0))
	}
	food, err := s.AddSheet("Food costs")
	if err != nil {
		t.Fatal(err)
	}
	food.Set(0, 0, "12")
	s.Set(0, 2, "='Food costs'!A1")
	if s.Display(0, 2) != "12" {
		t.Fatalf("quoted %s", s.Display(0, 2))
	}
	if _, err := s.AddSheet("income"); err == nil {
		t.Fatal("duplicate sheet name")
	}
	if err := income.Rename("Pay"); err != nil {
		t.Fatal(err)
	}
	if s.Source(1, 0) != "=SUM(Pay!B1:B2)" || s.Display(1, 0) != "4150" {
		t.Fatalf("renamed %q %s", s.Source(1, 0), s.Display(1, 0))
	}
	s.Set(0, 3, "=Pay!A2")
	income.Set(0, 1, "=Sheet1!A4")
	if s.Display(0, 3) != "#CYCLE" {
		t.Fatalf("cycle %s", s.Display(0, 3))
	}
	dir := t.TempDir()
	s.Path = filepath.Join(dir, "budget.xlsx")
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	if income.Path != s.Path || income.Dirty {
		t.Fatal("sheets do not share the save")
	}
	raw, err := os.ReadFile(s.Path)
	if err != nil {
		t.Fatal(err)
	}
	names := zipNames(t, raw)
	if !names["xl/worksheets/sheet1.xml"] || !names["xl/worksheets/sheet2.xml"] || !names["xl/worksheets/sheet3.xml"] {
		t.Fatalf("parts %v", names)
	}
	loaded, err := Open(s.Path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.SheetCount() != 3 || loaded.Name != "Sheet1" || loaded.SheetAt(1).Name != "Pay" {
		t.Fatalf("loaded %d %s %s", loaded.SheetCount(), loaded.Name, loaded.SheetAt(1).Name)
	}
	if loaded.Source(1, 0) != "=SUM(Pay!B1:B2)" || loaded.Display(1, 0) != "4150" {
		t.Fatalf("reloaded %q %s", loaded.Source(1, 0), loaded.Display(1, 0))
	}
	if loaded.SheetAt(2).Source(0, 0) != "12" || loaded.Display(0, 2) != "12" {
		t.Fatal("food sheet")
	}
}

func zipNames(t *testing.T, data []byte) map[string]bool {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]bool{}
	for _, f := range r.File {
		out[f.Name] = true
	}
	return out
}

func zipSheetXML(t *testing.T, data []byte) string {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range r.File {
		if strings.HasSuffix(f.Name, "sheet1.xml") {
			rc, err := f.Open()
			if err != nil {
				t.Fatal(err)
			}
			defer rc.Close()
			body, err := io.ReadAll(rc)
			if err != nil {
				t.Fatal(err)
			}
			return string(body)
		}
	}
	t.Fatal("sheet1.xml missing")
	return ""
}

func zipParts(t *testing.T, parts map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range parts {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(w, body); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestOpenMissingPathIsBlank(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new.csv")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "new.xlsx")
	if s.Path != want || s.Dirty || s.Source(0, 0) != "" {
		t.Fatalf("path %q dirty %v", s.Path, s.Dirty)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("csv was created")
	}
	if _, err := os.Stat(want + ".xlsx"); !os.IsNotExist(err) {
		t.Fatal("doubled xlsx")
	}
	if _, err := Open(t.TempDir()); err == nil {
		t.Fatal("directory opened")
	}
}

func TestOpenCreatesXLSXOnce(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "budget"))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "budget.xlsx")
	if s.Path != want || s.Dirty {
		t.Fatalf("path %q dirty %v", s.Path, s.Dirty)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatal(err)
	}
	again, err := Open(want)
	if err != nil {
		t.Fatal(err)
	}
	if again.Path != want {
		t.Fatal(again.Path)
	}
	if _, err := os.Stat(want + ".xlsx"); !os.IsNotExist(err) {
		t.Fatal("doubled xlsx")
	}
	upper := filepath.Join(dir, "Notes.XLSX")
	u, err := Open(upper)
	if err != nil {
		t.Fatal(err)
	}
	if u.Path != upper {
		t.Fatal(u.Path)
	}
	if _, err := os.Stat(upper + ".xlsx"); !os.IsNotExist(err) {
		t.Fatal("doubled upper xlsx")
	}
}

func TestColNames(t *testing.T) {
	if ColName(0) != "A" || ColName(25) != "Z" || ColName(26) != "AA" {
		t.Fatalf("names %s %s %s", ColName(0), ColName(25), ColName(26))
	}
	for _, name := range []string{"A", "Z", "AA", "b7"} {
		c, ok := ColIndex(strings.TrimRight(name, "0123456789"))
		if !ok {
			t.Fatal(name)
		}
		if strings.ToUpper(ColName(c)) != strings.ToUpper(strings.TrimRight(name, "0123456789")) {
			t.Fatalf("%s -> %d -> %s", name, c, ColName(c))
		}
	}
}

func mustAddr(t *testing.T, addr string) (int, int, bool) {
	t.Helper()
	c, r, ok := parseAddr(addr)
	if !ok {
		t.Fatalf("parse %s", addr)
	}
	return c, r, true
}

func parseAddr(addr string) (int, int, bool) {
	p := &parser{s: addr}
	ex, err := p.parseIdent()
	if err != nil {
		return 0, 0, false
	}
	cell, ok := ex.(cellExpr)
	if !ok {
		return 0, 0, false
	}
	p.skip()
	if p.i != len(p.s) {
		return 0, 0, false
	}
	return cell.c, cell.r, true
}

func TestInsertRowShiftsCellsAndFormulas(t *testing.T) {
	s := Blank()
	s.Set(0, 0, "Rent")
	s.Set(1, 0, "10")
	s.Set(0, 1, "Food")
	s.Set(1, 1, "20")
	s.Set(0, 2, "Transit")
	s.Set(1, 2, "30")
	s.Set(1, 3, "=SUM(B1:B3)")
	s.Set(2, 3, "note B2")
	s.Set(3, 0, "=$B$2")
	rows := s.Rows
	if !s.InsertRow(1) {
		t.Fatal("insert")
	}
	if s.Rows != rows+1 {
		t.Fatalf("rows %d", s.Rows)
	}
	if s.Source(0, 0) != "Rent" || s.Source(1, 0) != "10" || s.Source(0, 1) != "" || s.Source(1, 1) != "" {
		t.Fatalf("top %q %q blank %q %q", s.Source(0, 0), s.Source(1, 0), s.Source(0, 1), s.Source(1, 1))
	}
	if s.Source(0, 2) != "Food" || s.Source(1, 2) != "20" || s.Source(0, 3) != "Transit" || s.Source(1, 3) != "30" {
		t.Fatalf("shifted food %q %q transit %q %q", s.Source(0, 2), s.Source(1, 2), s.Source(0, 3), s.Source(1, 3))
	}
	if s.Source(1, 4) != "=SUM(B1:B4)" || s.Display(1, 4) != "60" {
		t.Fatalf("sum %q display %s", s.Source(1, 4), s.Display(1, 4))
	}
	if s.Source(2, 4) != "note B2" {
		t.Fatalf("text %q", s.Source(2, 4))
	}
	if s.Source(3, 0) != "=$B$3" || s.Display(3, 0) != "20" {
		t.Fatalf("abs %q display %s", s.Source(3, 0), s.Display(3, 0))
	}
}

func TestInsertRowGrowsATotalDirectlyBelow(t *testing.T) {
	s := Blank()
	s.Set(1, 0, "10")
	s.Set(1, 1, "20")
	s.Set(1, 2, "=SUM(B1:B2)")
	if !s.InsertRow(2) {
		t.Fatal("insert")
	}
	if s.Source(1, 0) != "10" || s.Source(1, 1) != "20" || s.Source(1, 2) != "" {
		t.Fatalf("body %q %q %q", s.Source(1, 0), s.Source(1, 1), s.Source(1, 2))
	}
	if s.Source(1, 3) != "=SUM(B1:B3)" || s.Display(1, 3) != "30" {
		t.Fatalf("total %q display %s", s.Source(1, 3), s.Display(1, 3))
	}

	far := Blank()
	far.Set(1, 0, "10")
	far.Set(1, 1, "20")
	far.Set(1, 4, "=SUM(B1:B2)")
	if !far.InsertRow(3) {
		t.Fatal("far insert")
	}
	if far.Source(1, 5) != "=SUM(B1:B2)" || far.Display(1, 5) != "30" {
		t.Fatalf("far %q display %s", far.Source(1, 5), far.Display(1, 5))
	}
}

func TestInsertRowUpdatesOtherSheets(t *testing.T) {
	s := Blank()
	if err := s.Rename("Food costs"); err != nil {
		t.Fatal(err)
	}
	other, err := s.AddSheet("Other")
	if err != nil {
		t.Fatal(err)
	}
	s.Set(1, 0, "10")
	s.Set(1, 2, "30")
	s.Set(2, 0, "=SUM('Food costs'!B1:B3)")
	other.Set(0, 0, "='Food costs'!B1")
	other.Set(0, 1, "=SUM('Food costs'!B1:B3)")
	other.Set(1, 0, "=A1")
	if !s.InsertRow(1) {
		t.Fatal("insert")
	}
	if s.Source(1, 0) != "10" || s.Source(1, 3) != "30" {
		t.Fatalf("values %q %q", s.Source(1, 0), s.Source(1, 3))
	}
	if s.Source(2, 0) != "=SUM('Food costs'!B1:B4)" || s.Display(2, 0) != "40" {
		t.Fatalf("local %q display %s", s.Source(2, 0), s.Display(2, 0))
	}
	if other.Source(0, 0) != "='Food costs'!B1" || other.Display(0, 0) != "10" {
		t.Fatalf("above %q display %s", other.Source(0, 0), other.Display(0, 0))
	}
	if other.Source(0, 1) != "=SUM('Food costs'!B1:B4)" || other.Display(0, 1) != "40" {
		t.Fatalf("other %q display %s", other.Source(0, 1), other.Display(0, 1))
	}
	if other.Source(1, 0) != "=A1" {
		t.Fatalf("local other %q", other.Source(1, 0))
	}
}

func TestInsertRowStopsAtTheEdge(t *testing.T) {
	s := Blank()
	if !s.GrowTo(0, MaxRow-1) {
		t.Fatal("grow")
	}
	s.Set(0, MaxRow-1, "end")
	if s.InsertRow(0) {
		t.Fatal("cell fell off")
	}
	if s.Source(0, MaxRow-1) != "end" || s.Source(0, 0) != "" {
		t.Fatalf("moved %q %q", s.Source(0, 0), s.Source(0, MaxRow-1))
	}

	s = Blank()
	s.Set(0, 0, "=A200")
	s.Set(1, 0, "keep")
	if s.InsertRow(0) {
		t.Fatal("ref fell off")
	}
	if s.Source(0, 0) != "=A200" || s.Source(1, 0) != "keep" {
		t.Fatalf("rewrote %q %q", s.Source(0, 0), s.Source(1, 0))
	}

	s = Blank()
	s.Set(0, 1, "=A199")
	if !s.InsertRow(0) {
		t.Fatal("near edge")
	}
	if s.Source(0, 2) != "=A200" {
		t.Fatalf("near %q", s.Source(0, 2))
	}
}
