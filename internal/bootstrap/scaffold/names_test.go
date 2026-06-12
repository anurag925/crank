package scaffold

import "testing"

func TestNewResource(t *testing.T) {
	cases := []struct {
		in   string
		want Resource
	}{
		{
			in: "Order",
			want: Resource{
				Pascal: "Order", PascalPlural: "Orders",
				Camel: "order", CamelPlural: "orders",
				Snake: "order", SnakePlural: "orders",
				Kebab: "order", KebabPlural: "orders",
				Receiver: "o",
			},
		},
		{
			in: "orders", // plural input is normalized to singular base
			want: Resource{
				Pascal: "Order", PascalPlural: "Orders",
				Camel: "order", CamelPlural: "orders",
				Snake: "order", SnakePlural: "orders",
				Kebab: "order", KebabPlural: "orders",
				Receiver: "o",
			},
		},
		{
			in: "OrderItem",
			want: Resource{
				Pascal: "OrderItem", PascalPlural: "OrderItems",
				Camel: "orderItem", CamelPlural: "orderItems",
				Snake: "order_item", SnakePlural: "order_items",
				Kebab: "order-item", KebabPlural: "order-items",
				Receiver: "o",
			},
		},
		{
			in: "order_items",
			want: Resource{
				Pascal: "OrderItem", PascalPlural: "OrderItems",
				Camel: "orderItem", CamelPlural: "orderItems",
				Snake: "order_item", SnakePlural: "order_items",
				Kebab: "order-item", KebabPlural: "order-items",
				Receiver: "o",
			},
		},
		{
			in: "category", // consonant + y pluralization
			want: Resource{
				Pascal: "Category", PascalPlural: "Categories",
				Camel: "category", CamelPlural: "categories",
				Snake: "category", SnakePlural: "categories",
				Kebab: "category", KebabPlural: "categories",
				Receiver: "c",
			},
		},
		{
			in: "box", // -x pluralization
			want: Resource{
				Pascal: "Box", PascalPlural: "Boxes",
				Camel: "box", CamelPlural: "boxes",
				Snake: "box", SnakePlural: "boxes",
				Kebab: "box", KebabPlural: "boxes",
				Receiver: "b",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got := NewResource(c.in)
			if got != c.want {
				t.Errorf("NewResource(%q)\n got: %+v\nwant: %+v", c.in, got, c.want)
			}
		})
	}
}

func TestParseFields(t *testing.T) {
	fields, err := ParseFields([]string{"title:string", "price:float", "in_stock:bool", "ship_at:time"})
	if err != nil {
		t.Fatalf("ParseFields: %v", err)
	}
	if len(fields) != 4 {
		t.Fatalf("expected 4 fields, got %d", len(fields))
	}

	if fields[0].GoName != "Title" || fields[0].GoType != "string" || fields[0].SQLType != "TEXT" {
		t.Errorf("title field wrong: %+v", fields[0])
	}
	if fields[1].GoType != "float64" || fields[1].SQLType != "DOUBLE PRECISION" {
		t.Errorf("price field wrong: %+v", fields[1])
	}
	if fields[2].GoType != "bool" || fields[2].Validate != "" {
		t.Errorf("bool field should have no required validation: %+v", fields[2])
	}
	if !fields[3].IsTime || fields[3].GoType != "time.Time" {
		t.Errorf("time field wrong: %+v", fields[3])
	}
	if !hasTimeField(fields) {
		t.Error("hasTimeField should be true when a time field is present")
	}
}

func TestParseFieldsDefaultsToString(t *testing.T) {
	fields, err := ParseFields([]string{"name"})
	if err != nil {
		t.Fatalf("ParseFields: %v", err)
	}
	if fields[0].GoType != "string" {
		t.Errorf("expected default string type, got %q", fields[0].GoType)
	}
}

func TestParseFieldsUnknownType(t *testing.T) {
	if _, err := ParseFields([]string{"foo:widget"}); err == nil {
		t.Fatal("expected error for unknown field type")
	}
}
