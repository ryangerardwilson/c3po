package sheet

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	blankCols = 8
	blankRows = 24
	MaxCol    = 40
	MaxRow    = 200
)

type Addr struct {
	C int
	R int
}

type Sheet struct {
	Name  string
	Cols  int
	Rows  int
	Path  string
	Dirty bool

	book  *Book
	cells map[Addr]string
	epoch uint64
	cache map[Addr]cached
}

type cached struct {
	epoch uint64
	v     Value
}

type Stats struct {
	Count   int
	Sum     float64
	Average float64
	Median  float64
	Mode    float64
	HasMode bool
}

func Blank() *Sheet {
	b := &Book{}
	s := emptyGrid()
	s.Name = "Sheet1"
	s.book = b
	b.Sheets = []*Sheet{s}
	return s
}

func Open(path string) (*Sheet, error) {
	if strings.TrimSpace(path) == "" {
		return Blank(), nil
	}
	path = expandPath(strings.TrimSpace(path))
	givenExists, err := usableFile(path)
	if err != nil {
		return nil, err
	}
	target := xlsxPath(path)
	if target != path {
		targetExists, err := usableFile(target)
		if err != nil {
			return nil, err
		}
		if targetExists {
			return readWorkbook(target)
		}
	} else if givenExists {
		return readWorkbook(path)
	}
	if givenExists {
		s, err := readWorkbook(path)
		if err != nil {
			return nil, err
		}
		s.setPath(target)
		if err := s.Save(); err != nil {
			return nil, err
		}
		return s, nil
	}
	s := Blank()
	s.setPath(target)
	if err := s.Save(); err != nil {
		return nil, err
	}
	return s, nil
}

func usableFile(path string) (bool, error) {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if info.IsDir() {
		return false, fmt.Errorf("%s is a directory", path)
	}
	return true, nil
}

func readWorkbook(path string) (*Sheet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s *Sheet
	if isXLSX(data) {
		s, err = ParseXLSX(data)
	} else {
		s, err = Parse(data)
	}
	if err != nil {
		return nil, err
	}
	s.setPath(path)
	s.setClean()
	return s, nil
}

func Parse(data []byte) (*Sheet, error) {
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	s := Blank()
	if len(bytes.TrimSpace(data)) == 0 {
		return s, nil
	}
	reader := csv.NewReader(bytes.NewReader(data))
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read csv: %w", err)
	}
	for r, record := range records {
		for c, field := range record {
			if field == "" {
				continue
			}
			if c >= MaxCol || r >= MaxRow {
				return nil, fmt.Errorf("csv exceeds %d columns or %d rows", MaxCol, MaxRow)
			}
			s.cells[Addr{C: c, R: r}] = field
		}
	}
	s.ensureRoom()
	s.Dirty = false
	return s, nil
}

func (s *Sheet) Save() error {
	if strings.TrimSpace(s.Path) == "" {
		return errors.New("no path")
	}
	path := xlsxPath(s.Path)
	data, err := s.MarshalXLSX()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if dir == "" {
		dir = "."
	}
	f, err := os.CreateTemp(dir, ".xl-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	_, werr := f.Write(data)
	cerr := f.Close()
	if werr != nil || cerr != nil {
		os.Remove(tmp)
		if werr != nil {
			return werr
		}
		return cerr
	}
	if err := os.Chmod(tmp, 0o644); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	s.setPath(path)
	s.setClean()
	return nil
}

func (s *Sheet) SaveAs(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("no path")
	}
	s.setPath(xlsxPath(expandPath(path)))
	return s.Save()
}

func (s *Sheet) Source(c, r int) string {
	if s == nil || s.cells == nil {
		return ""
	}
	return s.cells[Addr{C: c, R: r}]
}

func (s *Sheet) Set(c, r int, src string) bool {
	if c < 0 || r < 0 || c >= s.Cols || r >= s.Rows {
		return false
	}
	a := Addr{C: c, R: r}
	if src == "" {
		if _, ok := s.cells[a]; !ok {
			return false
		}
		delete(s.cells, a)
	} else if s.cells[a] == src {
		return false
	} else {
		if s.cells == nil {
			s.cells = map[Addr]string{}
		}
		s.cells[a] = src
	}
	s.touch()
	return true
}

func (s *Sheet) GrowTo(c, r int) bool {
	if c < 0 || r < 0 || c >= MaxCol || r >= MaxRow {
		return false
	}
	if c >= s.Cols {
		s.Cols = c + 1
	}
	if r >= s.Rows {
		s.Rows = r + 1
	}
	return true
}

func (s *Sheet) Value(c, r int) Value {
	return s.eval(c, r, map[visit]bool{})
}

func (s *Sheet) Display(c, r int) string {
	src := strings.TrimSpace(s.Source(c, r))
	v := s.Value(c, r)
	if strings.HasPrefix(src, "=") && v.Kind == Empty {
		return "0"
	}
	return strings.ReplaceAll(v.String(), "\n", " ")
}

func (s *Sheet) Stats(c1, r1, c2, r2 int) Stats {
	if c1 > c2 {
		c1, c2 = c2, c1
	}
	if r1 > r2 {
		r1, r2 = r2, r1
	}
	var nums []float64
	for r := r1; r <= r2; r++ {
		for c := c1; c <= c2; c++ {
			src := strings.TrimSpace(s.Source(c, r))
			v := s.Value(c, r)
			if strings.HasPrefix(src, "=") && v.Kind == Empty {
				nums = append(nums, 0)
				continue
			}
			if v.Kind == Number {
				nums = append(nums, v.Num)
			}
		}
	}
	return statsFrom(nums)
}

// PureNumbers reports whether the rectangle holds at least one number and no
// text or errors. Blank cells are ignored.
func (s *Sheet) PureNumbers(c1, r1, c2, r2 int) bool {
	if s == nil {
		return false
	}
	if c1 > c2 {
		c1, c2 = c2, c1
	}
	if r1 > r2 {
		r1, r2 = r2, r1
	}
	n := 0
	for r := r1; r <= r2; r++ {
		for c := c1; c <= c2; c++ {
			src := strings.TrimSpace(s.Source(c, r))
			if src == "" {
				continue
			}
			v := s.Value(c, r)
			if strings.HasPrefix(src, "=") && v.Kind == Empty {
				n++
				continue
			}
			if v.Kind != Number {
				return false
			}
			n++
		}
	}
	return n > 0
}

func (st Stats) String() string {
	if st.Count == 0 {
		return "Sum -   Average -   Median -   Mode -   n 0"
	}
	mode := "none"
	if st.HasMode {
		mode = formatGrouped(st.Mode)
	}
	return fmt.Sprintf("Sum %s   Average %s   Median %s   Mode %s   n %d",
		formatGrouped(st.Sum), formatGrouped(st.Average), formatGrouped(st.Median), mode, st.Count)
}

func (s *Sheet) LastUsed() (c, r int, ok bool) {
	c, r = -1, -1
	for a, src := range s.cells {
		if strings.TrimSpace(src) == "" {
			continue
		}
		if a.R > r || (a.R == r && a.C > c) {
			c, r = a.C, a.R
		}
	}
	return c, r, c >= 0
}

func (s *Sheet) LastCol(row int) int {
	max := -1
	for a, src := range s.cells {
		if a.R == row && strings.TrimSpace(src) != "" && a.C > max {
			max = a.C
		}
	}
	if max < 0 {
		return 0
	}
	return max
}

func ColName(col int) string {
	if col < 0 {
		return "?"
	}
	n := col + 1
	var b [8]byte
	i := len(b)
	for n > 0 {
		n--
		i--
		b[i] = byte('A' + n%26)
		n /= 26
	}
	return string(b[i:])
}

func ColIndex(name string) (int, bool) {
	if name == "" {
		return 0, false
	}
	n := 0
	for i := 0; i < len(name); i++ {
		ch := name[i]
		if ch >= 'a' && ch <= 'z' {
			ch -= 'a' - 'A'
		}
		if ch < 'A' || ch > 'Z' {
			return 0, false
		}
		n = n*26 + int(ch-'A'+1)
		if n > 10000 {
			return 0, false
		}
	}
	return n - 1, true
}

func AddrLabel(c, r int) string {
	return ColName(c) + strconv.Itoa(r+1)
}

func RangeLabel(c1, r1, c2, r2 int) string {
	if c1 > c2 {
		c1, c2 = c2, c1
	}
	if r1 > r2 {
		r1, r2 = r2, r1
	}
	a := AddrLabel(c1, r1)
	b := AddrLabel(c2, r2)
	if a == b {
		return a
	}
	return a + ":" + b
}

func (s *Sheet) eval(c, r int, stack map[visit]bool) Value {
	if c < 0 || r < 0 {
		return errv("#REF")
	}
	a := Addr{C: c, R: r}
	key := visit{sheet: s.Name, a: a}
	if stack[key] {
		return errv("#CYCLE")
	}
	epoch := s.epoch
	if s.book != nil {
		epoch = s.book.epoch
	}
	if s.cache != nil {
		if hit, ok := s.cache[a]; ok && hit.epoch == epoch {
			return hit.v
		}
	}
	if c >= s.Cols || r >= s.Rows {
		return Value{}
	}
	stack[key] = true
	v := s.evalStored(s.cells[a], stack)
	delete(stack, key)
	if s.cache == nil {
		s.cache = map[Addr]cached{}
	}
	s.cache[a] = cached{epoch: epoch, v: v}
	return v
}

func (s *Sheet) evalStored(src string, stack map[visit]bool) Value {
	trim := strings.TrimSpace(src)
	if trim == "" {
		return Value{}
	}
	if strings.HasPrefix(trim, "=") {
		return evalFormula(trim, s, stack)
	}
	if n, ok := parseLiteralNumber(trim); ok {
		return num(n)
	}
	return txt(trim)
}

func (s *Sheet) walk(c1, r1, c2, r2 int, stack map[visit]bool) ([]float64, Value) {
	if c1 > c2 {
		c1, c2 = c2, c1
	}
	if r1 > r2 {
		r1, r2 = r2, r1
	}
	if c2 < 0 || r2 < 0 || c1 >= s.Cols || r1 >= s.Rows {
		return nil, Value{}
	}
	if c1 < 0 {
		c1 = 0
	}
	if r1 < 0 {
		r1 = 0
	}
	if c2 >= s.Cols {
		c2 = s.Cols - 1
	}
	if r2 >= s.Rows {
		r2 = s.Rows - 1
	}
	var nums []float64
	for r := r1; r <= r2; r++ {
		for c := c1; c <= c2; c++ {
			v := s.eval(c, r, stack)
			switch v.Kind {
			case Error:
				return nil, v
			case Number:
				nums = append(nums, v.Num)
			}
		}
	}
	return nums, Value{}
}

func (s *Sheet) used() (maxC, maxR int, ok bool) {
	maxC, maxR = -1, -1
	for a, src := range s.cells {
		if strings.TrimSpace(src) == "" {
			continue
		}
		if a.C > maxC {
			maxC = a.C
		}
		if a.R > maxR {
			maxR = a.R
		}
	}
	return maxC, maxR, maxC >= 0
}

func (s *Sheet) ensureRoom() {
	cols, rows := blankCols, blankRows
	if maxC, maxR, ok := s.used(); ok {
		if maxC+3 > cols {
			cols = maxC + 3
		}
		if maxR+6 > rows {
			rows = maxR + 6
		}
	}
	if cols > MaxCol {
		cols = MaxCol
	}
	if rows > MaxRow {
		rows = MaxRow
	}
	s.Cols = cols
	s.Rows = rows
}

func statsFrom(nums []float64) Stats {
	if len(nums) == 0 {
		return Stats{}
	}
	total := 0.0
	for _, n := range nums {
		total += n
	}
	mode, has := modeOf(nums)
	return Stats{
		Count:   len(nums),
		Sum:     total,
		Average: total / float64(len(nums)),
		Median:  medianOf(nums),
		Mode:    mode,
		HasMode: has,
	}
}

func parseLiteralNumber(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	body := s
	if body[0] == '+' || body[0] == '-' {
		body = body[1:]
		if body == "" {
			return 0, false
		}
	}
	if strings.Contains(body, ",") {
		if !thousandsOK(body) {
			return 0, false
		}
		s = strings.ReplaceAll(s, ",", "")
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

func thousandsOK(body string) bool {
	intPart, frac, _ := strings.Cut(body, ".")
	if strings.Contains(frac, ",") || !digits(frac) {
		return false
	}
	groups := strings.Split(intPart, ",")
	if len(groups) < 2 || len(groups[0]) < 1 || len(groups[0]) > 3 || !digits(groups[0]) {
		return false
	}
	for _, group := range groups[1:] {
		if len(group) != 3 || !digits(group) {
			return false
		}
	}
	return true
}

func digits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func expandPath(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		if path == "~" {
			return home
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// InsertRow inserts a blank row at at and shifts later rows down.
// References to this sheet move with those rows. A range grows when the
// new row is inside it or directly under it. InsertRow returns false when
// a cell or a reference would pass the last row.
func (s *Sheet) InsertRow(at int) bool {
	if s == nil || at < 0 || at >= MaxRow {
		return false
	}
	for addr := range s.cells {
		if addr.R >= at && addr.R+1 >= MaxRow {
			return false
		}
	}
	sheets := []*Sheet{s}
	if s.book != nil && len(s.book.Sheets) > 0 {
		sheets = s.book.Sheets
	}
	type rewrite struct {
		sh   *Sheet
		addr Addr
		src  string
	}
	var edits []rewrite
	for _, sh := range sheets {
		if sh == nil {
			continue
		}
		for addr, src := range sh.cells {
			next, ok := shiftFormulaRows(src, s.Name, sh == s, at)
			if !ok {
				return false
			}
			if next != src {
				edits = append(edits, rewrite{sh, addr, next})
			}
		}
	}
	for _, ed := range edits {
		ed.sh.cells[ed.addr] = ed.src
	}
	if len(s.cells) > 0 {
		next := make(map[Addr]string, len(s.cells))
		for addr, src := range s.cells {
			if addr.R >= at {
				addr.R++
			}
			next[addr] = src
		}
		s.cells = next
	}
	if s.Rows < MaxRow {
		s.Rows++
	}
	s.touch()
	return true
}
