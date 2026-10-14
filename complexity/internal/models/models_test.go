package models

import (
	"strings"
	"testing"
)

func TestPrismaDegreeIsNotM2M(t *testing.T) {
	src := `
model User {
  id    Int    @id
  posts Post[]
}
model Post {
  id       Int    @id
  author   User   @relation(fields: [authorId], references: [id])
  authorId Int
  title    String
}
`
	sec := Scan([]Source{{Path: "schema.prisma", Language: "prisma", Text: src}}, Thresholds{Degree: 10, Depth: 5, M2M: 4}, testConv())
	by := byName(sec)
	if by["User"].Degree != 2 || by["Post"].Degree != 2 {
		t.Fatalf("degree user=%d post=%d", by["User"].Degree, by["Post"].Degree)
	}
	if by["User"].M2M != 0 || by["Post"].M2M != 0 || by["User"].Cycles || by["Post"].Cycles {
		t.Fatalf("m2m/cycles %#v %#v", by["User"], by["Post"])
	}
}

func TestSQLColumnOrderAndCycle(t *testing.T) {
	bad := `
CREATE TABLE order_payment (
  id integer primary key,
  status text,
  customer_id integer references user_account(id),
  created_at timestamp
);
CREATE TABLE a_node (id int primary key, b_id int references b_node(id));
CREATE TABLE b_node (id int primary key, a_id int references a_node(id));
`
	sec := Scan([]Source{{Path: "m.sql", Language: "sql", Text: bad}}, Thresholds{Degree: 10, Depth: 5, M2M: 4}, testConv())
	pay := byName(sec)["order_payment"]
	if !has(pay.Violations, "column_order") {
		t.Fatalf("violations %#v", pay)
	}
	if len(pay.Suggestions) == 0 || !strings.Contains(pay.Suggestions[0], "customer_id") {
		t.Fatalf("suggestion %#v", pay.Suggestions)
	}
	if !byName(sec)["a_node"].Cycles || byName(sec)["a_node"].Depth != 1 {
		t.Fatalf("cycle depth=%d cycles=%v", byName(sec)["a_node"].Depth, byName(sec)["a_node"].Cycles)
	}
}

func TestPrismaNaming(t *testing.T) {
	src := "model Order {\n  id Int @id\n}\n"
	sec := Scan([]Source{{Path: "schema.prisma", Language: "prisma", Text: src}}, Thresholds{Degree: 10}, testConv())
	m := byName(sec)["Order"]
	if !has(m.Violations, "naming") {
		t.Fatalf("%#v", m)
	}
	if len(m.Suggestions) == 0 || !strings.Contains(m.Suggestions[0], "order_core") {
		t.Fatalf("%#v", m.Suggestions)
	}
}

func TestPythonAndGORM(t *testing.T) {
	py := `
class Order(Base):
    __tablename__ = "order_payment"
    id = Column(Integer, primary_key=True)
    customer_id = Column(Integer, ForeignKey("user_account.id"))
    status = Column(String)
    customer = relationship("User")
`
	goSrc := `
package p
type Order struct {
    ID uint ` + "`gorm:\"primaryKey\"`" + `
    CustomerID uint
    Customer User ` + "`gorm:\"foreignKey:CustomerID\"`" + `
    Status string
    CreatedAt time.Time
}
`
	sec := Scan([]Source{
		{Path: "models.py", Language: "python", Text: py},
		{Path: "models.go", Language: "go", Text: goSrc},
	}, Thresholds{Degree: 10, Depth: 5, M2M: 4}, testConv())
	pyModel := fileModel(sec, "models.py")
	goModel := fileModel(sec, "models.go")
	if has(pyModel.Violations, "naming") || has(pyModel.Violations, "column_order") {
		t.Fatalf("python %#v", pyModel)
	}
	if goModel.FanOut != 1 || has(goModel.Violations, "column_order") {
		t.Fatalf("gorm %#v", goModel)
	}
}

func fileModel(sec Section, file string) Model {
	for _, m := range sec.Models {
		if m.File == file {
			return m
		}
	}
	return Model{}
}

func testConv() Conventions {
	return Conventions{Naming: true, ColumnOrder: true, DataColumnName: "data", LogSuffix: "logs", RequireFeatureMapping: true}
}

func byName(sec Section) map[string]Model {
	out := map[string]Model{}
	for _, m := range sec.Models {
		out[m.Name] = m
	}
	return out
}

func has(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
