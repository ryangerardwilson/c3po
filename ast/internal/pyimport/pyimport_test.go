package pyimport

import "testing"

func TestFromOneLine(t *testing.T) {
	imp, ok := From("from pkg.mod import a, b as c  # noqa")
	if !ok || imp.Module != "pkg.mod" || len(imp.Names) != 2 {
		t.Fatalf("%+v", imp)
	}
	if imp.Names[1].Orig != "b" || imp.Names[1].Local != "c" {
		t.Fatalf("%+v", imp.Names)
	}
}

func TestScanParenthesized(t *testing.T) {
	src := "from pkg import (\n    alpha,\n    beta as b,  # why\n)\nx = 1\n"
	imps := Scan(src)
	if len(imps) != 1 || len(imps[0].Names) != 2 {
		t.Fatalf("%+v", imps)
	}
	if imps[0].Names[0] != (Name{Orig: "alpha", Local: "alpha", Line: 2}) || imps[0].Names[1].Line != 3 {
		t.Fatalf("%+v", imps[0].Names)
	}
}

func TestScanBackslash(t *testing.T) {
	imps := Scan("from pkg import a, \\\n    b\n")
	if len(imps) != 1 || len(imps[0].Names) != 2 || imps[0].Names[1].Line != 2 {
		t.Fatalf("%+v", imps)
	}
}

func TestScanPlainImport(t *testing.T) {
	imps := Scan("import pkg.module, os as o\n")
	if len(imps) != 2 || imps[0].Module != "pkg.module" || imps[0].From || imps[0].Names[0].Local != "pkg" {
		t.Fatalf("%+v", imps)
	}
	if imps[1].Names[0].Local != "o" {
		t.Fatalf("%+v", imps[1])
	}
}

func TestScanSkipsStringsAndComments(t *testing.T) {
	src := "\"from pkg import quoted\"\n# from pkg import comment\ndoc = \"\"\"\nfrom pkg import inside\n\"\"\"\n'''\nimport pkg.hidden\n'''\nfrom pkg import real\n"
	imps := Scan(src)
	if len(imps) != 1 || imps[0].Names[0].Orig != "real" || imps[0].Names[0].Line != 9 {
		t.Fatalf("%+v", imps)
	}
}
