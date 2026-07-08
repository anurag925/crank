package seedgen

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// fakeValue generates a SQL-compatible literal value string for a field,
// using the field's type and name to produce realistic-looking data.
func fakeValue(fi FieldInfo) string {
	if fi.IsEnum && len(fi.EnumValues) > 0 {
		return escapeSQLString(fi.EnumValues[rand.Intn(len(fi.EnumValues))])
	}

	// Check ORM type hint first (e.g. `bun:"type:uuid"`).
	ormLower := strings.ToLower(fi.ORMType)
	if strings.Contains(ormLower, "uuid") {
		return "'" + fakeUUID() + "'"
	}

	typ := strings.TrimPrefix(fi.GoType, "*") // handle nullable types
	switch {
	case strings.HasSuffix(typ, "UUID") || strings.HasSuffix(typ, "uuid"):
		return "'" + fakeUUID() + "'"
	case typ == "string" || typ == "String":
		return "'" + fakeStringForField(fi.Name, fi.ColumnName) + "'"
	case typ == "int" || typ == "int8" || typ == "int16" || typ == "int32" || typ == "int64":
		return fmt.Sprintf("%d", rand.Intn(99999)+1)
	case typ == "uint" || typ == "uint8" || typ == "uint16" || typ == "uint32" || typ == "uint64":
		return fmt.Sprintf("%d", rand.Intn(99999)+1)
	case typ == "float32" || typ == "float64":
		return fmt.Sprintf("%.2f", rand.Float64()*1000)
	case typ == "bool":
		if rand.Intn(2) == 0 {
			return "true"
		}
		return "false"
	case typ == "Time" || strings.HasSuffix(typ, "Time"):
		return "'" + fakeTime() + "'"
	default:
		return "'" + escapeSQLString(fmt.Sprintf("fake-%s", fi.Name)) + "'"
	}
}

// fakeStringForField generates a context-aware random string based on the
// field name. This produces more realistic seed data than generic lorem ipsum.
func fakeStringForField(fieldName, columnName string) string {
	name := strings.ToLower(fieldName)
	col := strings.ToLower(columnName)
	switch {
	// Check column name first (more reliable for ORM-mapped fields).
	case col == "id" || strings.HasSuffix(col, "_id") || strings.HasSuffix(col, "_uuid") ||
		name == "id" || strings.HasSuffix(name, "_id") || strings.HasSuffix(name, "_uuid"):
		return fakeUUID()
	case strings.Contains(name, "email"):
		return fakeEmail()
	case strings.Contains(name, "phone") || strings.Contains(name, "mobile") || strings.Contains(name, "tel"):
		return fakePhone()
	case strings.Contains(name, "url") || strings.Contains(name, "website") || strings.Contains(name, "link"):
		return fakeURL()
	case strings.Contains(name, "address") || strings.Contains(name, "street") || strings.Contains(name, "city"):
		return fakeAddress()
	case strings.Contains(name, "first_name") || strings.Contains(name, "firstname"):
		return fakeFirstName()
	case strings.Contains(name, "last_name") || strings.Contains(name, "lastname"):
		return fakeLastName()
	case strings.Contains(name, "name"):
		return fakeFullName()
	case strings.Contains(name, "title") || strings.Contains(name, "subject"):
		return fakeTitle()
	case strings.Contains(name, "description") || strings.Contains(name, "desc"):
		return fakeSentence()
	case strings.Contains(name, "password") || strings.Contains(name, "secret"):
		return "password123"
	case strings.Contains(name, "token") || strings.Contains(name, "key"):
		return fakeToken()
	case strings.Contains(name, "status"):
		return fakeStatus()
	case strings.Contains(name, "code") || strings.Contains(name, "sku"):
		return fakeCode()
	case strings.Contains(name, "color") || strings.Contains(name, "colour"):
		return fakeColor()
	default:
		return fakeFirstName() // sensible default
	}
}

var firstNames = []string{
	"Alice", "Bob", "Charlie", "Diana", "Eve", "Frank", "Grace",
	"Hank", "Ivy", "Jack", "Kate", "Leo", "Mia", "Noah",
	"Olivia", "Paul", "Quinn", "Rose", "Sam", "Tina",
}

var lastNames = []string{
	"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia",
	"Miller", "Davis", "Rodriguez", "Martinez", "Taylor", "Thomas",
}

var domains = []string{
	"example.com", "test.org", "demo.io", "mailinator.com",
	"acmecorp.co", "startup.dev",
}

var colors = []string{
	"red", "blue", "green", "yellow", "purple", "orange",
	"pink", "brown", "black", "white", "gray", "teal",
}

func fakeFirstName() string { return firstNames[rand.Intn(len(firstNames))] }
func fakeLastName() string  { return lastNames[rand.Intn(len(lastNames))] }
func fakeFullName() string  { return fakeFirstName() + " " + fakeLastName() }

func fakeEmail() string {
	return strings.ToLower(fakeFirstName() + "." + fakeLastName() + "@" + domains[rand.Intn(len(domains))])
}

func fakePhone() string {
	return fmt.Sprintf("+1-%03d-%03d-%04d", rand.Intn(999)+1, rand.Intn(999), rand.Intn(9999))
}

func fakeURL() string {
	return fmt.Sprintf("https://www.%s.%s", strings.ToLower(fakeFirstName()), domains[rand.Intn(len(domains))])
}

func fakeAddress() string {
	return fmt.Sprintf("%d %s %s", rand.Intn(9999)+1, fakeLastName(), rdPick("St", "Ave", "Blvd", "Dr", "Ln"))
}

func fakeTitle() string {
	return rdPick("Senior", "Junior", "Lead", "Principal", "Associate") + " " +
		rdPick("Software Engineer", "Product Manager", "Designer", "Data Scientist", "DevOps Engineer")
}

func fakeSentence() string {
	return rdPick("Lorem ipsum dolor sit amet", "Consectetur adipiscing elit",
		"Sed do eiusmod tempor incididunt", "Ut labore et dolore magna aliqua",
		"Ut enim ad minim veniam", "Duis aute irure dolor in reprehenderit")
}

func fakeToken() string {
	return fmt.Sprintf("tok_%s%s", fakeUUID()[:8], fakeUUID()[:8])
}

func fakeStatus() string {
	return rdPick("active", "inactive", "pending", "archived", "draft")
}

func fakeCode() string {
	return strings.ToUpper(fakeFirstName()[:3] + fmt.Sprintf("%04d", rand.Intn(9999)))
}

func fakeColor() string {
	return colors[rand.Intn(len(colors))]
}

// fakeUUID generates a random UUID v4 string.
func fakeUUID() string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		rand.Uint32(), rand.Intn(0xffff), rand.Intn(0xffff),
		rand.Intn(0xffff), rand.Int63n(0xffffffffffff))
}

// fakeTime generates a recent SQL-compatible timestamp string.
func fakeTime() string {
	t := time.Now().UTC().Add(-time.Duration(rand.Intn(90*24)) * time.Hour)
	return t.Format("2006-01-02 15:04:05")
}

// escapeSQLString wraps a string in single quotes after escaping any
// single quotes within it.
func escapeSQLString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func rdPick(items ...string) string {
	return items[rand.Intn(len(items))]
}
