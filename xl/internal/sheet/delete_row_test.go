package sheet

import "testing"

func TestDeleteRowShiftsCellsAndFormulas(t *testing.T) {
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
	if !s.DeleteRow(1) {
		t.Fatal("delete")
	}
	if s.Rows != rows-1 {
		t.Fatalf("rows %d", s.Rows)
	}
	if s.Source(0, 0) != "Rent" || s.Source(1, 0) != "10" || s.Source(0, 1) != "Transit" || s.Source(1, 1) != "30" {
		t.Fatalf("shifted %q %q %q %q", s.Source(0, 0), s.Source(1, 0), s.Source(0, 1), s.Source(1, 1))
	}
	if s.Source(0, 2) != "" || s.Source(1, 2) != "=SUM(B1:B2)" || s.Display(1, 2) != "40" {
		t.Fatalf("sum %q display %s blank %q", s.Source(1, 2), s.Display(1, 2), s.Source(0, 2))
	}
	if s.Source(2, 2) != "note B2" {
		t.Fatalf("text %q", s.Source(2, 2))
	}
	if s.Source(3, 0) != "=#REF!" || s.Display(3, 0) != "#REF" {
		t.Fatalf("ref %q display %s", s.Source(3, 0), s.Display(3, 0))
	}
}

func TestDeleteRowShrinksATotalAndLeavesAFarRange(t *testing.T) {
	s := Blank()
	s.Set(1, 0, "10")
	s.Set(1, 1, "20")
	s.Set(1, 2, "=SUM(B1:B2)")
	if !s.DeleteRow(0) {
		t.Fatal("delete")
	}
	if s.Source(1, 0) != "20" || s.Source(1, 1) != "=SUM(B1:B1)" || s.Display(1, 1) != "20" {
		t.Fatalf("total %q %q display %s", s.Source(1, 0), s.Source(1, 1), s.Display(1, 1))
	}

	far := Blank()
	far.Set(1, 0, "10")
	far.Set(1, 1, "20")
	far.Set(1, 4, "=SUM(B1:B2)")
	if !far.DeleteRow(3) {
		t.Fatal("far delete")
	}
	if far.Source(1, 3) != "=SUM(B1:B2)" || far.Display(1, 3) != "30" {
		t.Fatalf("far %q display %s", far.Source(1, 3), far.Display(1, 3))
	}
}

func TestDeleteRowUpdatesOtherSheets(t *testing.T) {
	s := Blank()
	if err := s.Rename("Food costs"); err != nil {
		t.Fatal(err)
	}
	other, err := s.AddSheet("Other")
	if err != nil {
		t.Fatal(err)
	}
	s.Set(1, 0, "10")
	s.Set(1, 1, "20")
	s.Set(1, 2, "30")
	other.Set(0, 0, "='Food costs'!B2")
	other.Set(0, 1, "=SUM('Food costs'!B1:B3)")
	other.Set(1, 0, "=A1")
	if !s.DeleteRow(1) {
		t.Fatal("delete")
	}
	if s.Source(1, 0) != "10" || s.Source(1, 1) != "30" {
		t.Fatalf("values %q %q", s.Source(1, 0), s.Source(1, 1))
	}
	if other.Source(0, 0) != "=#REF!" || other.Display(0, 0) != "#REF" {
		t.Fatalf("exact %q display %s", other.Source(0, 0), other.Display(0, 0))
	}
	if other.Source(0, 1) != "=SUM('Food costs'!B1:B2)" || other.Display(0, 1) != "40" {
		t.Fatalf("range %q display %s", other.Source(0, 1), other.Display(0, 1))
	}
	if other.Source(1, 0) != "=A1" || other.Display(1, 0) != "#REF" {
		t.Fatalf("local %q display %s", other.Source(1, 0), other.Display(1, 0))
	}
}

func TestDeleteRowRejectsAnAbsentRow(t *testing.T) {
	s := Blank()
	s.Set(0, 0, "keep")
	if s.DeleteRow(-1) || s.DeleteRow(s.Rows) {
		t.Fatal("accepted")
	}
	if s.Source(0, 0) != "keep" || s.Rows != blankRows {
		t.Fatalf("changed %q rows %d", s.Source(0, 0), s.Rows)
	}
}
