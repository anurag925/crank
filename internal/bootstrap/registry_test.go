package bootstrap

import (
	"embed"
	"testing"
)

// testFeature is a minimal Feature implementation for registry tests.
type testFeature struct {
	name string
}

func (f testFeature) Name() string           { return f.name }
func (f testFeature) Description() string    { return "test feature " + f.name }
func (f testFeature) Files() []FileMapping   { return nil }
func (f testFeature) Templates() embed.FS    { return embed.FS{} }
func (f testFeature) Dependencies() []string { return nil }
func (f testFeature) Requirements() []string { return nil }

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	f := testFeature{name: "base"}
	if err := reg.Register(f); err != nil {
		t.Fatalf("Register: %v", err)
	}
	got, ok := reg.Get("base")
	if !ok {
		t.Fatal("Get returned false for registered feature")
	}
	if got.Name() != "base" {
		t.Errorf("Name() = %q, want %q", got.Name(), "base")
	}
}

func TestRegistry_DuplicateRegister(t *testing.T) {
	reg := NewRegistry()
	f := testFeature{name: "base"}
	if err := reg.Register(f); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	if err := reg.Register(f); err == nil {
		t.Fatal("expected error on duplicate registration")
	}
}

func TestRegistry_GetUnknown(t *testing.T) {
	reg := NewRegistry()
	_, ok := reg.Get("nope")
	if ok {
		t.Fatal("Get should return false for unknown feature")
	}
}

func TestRegistry_MustRegister_PanicsOnDuplicate(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("MustRegister should panic on duplicate")
		}
	}()
	reg := NewRegistry()
	f := testFeature{name: "base"}
	reg.MustRegister(f)
	reg.MustRegister(f) // should panic
}

func TestRegistry_Names_PreservesOrder(t *testing.T) {
	reg := NewRegistry()
	reg.MustRegister(testFeature{name: "alpha"})
	reg.MustRegister(testFeature{name: "beta"})
	reg.MustRegister(testFeature{name: "gamma"})

	names := reg.Names()
	want := []string{"alpha", "beta", "gamma"}
	if len(names) != len(want) {
		t.Fatalf("len = %d, want %d", len(names), len(want))
	}
	for i, n := range names {
		if n != want[i] {
			t.Errorf("Names[%d] = %q, want %q", i, n, want[i])
		}
	}
}

func TestRegistry_All_PreservesOrder(t *testing.T) {
	reg := NewRegistry()
	reg.MustRegister(testFeature{name: "a"})
	reg.MustRegister(testFeature{name: "b"})

	all := reg.All()
	if len(all) != 2 {
		t.Fatalf("len = %d, want 2", len(all))
	}
	if all[0].Name() != "a" || all[1].Name() != "b" {
		t.Errorf("All() order = [%s, %s], want [a, b]", all[0].Name(), all[1].Name())
	}
}

func TestResolve(t *testing.T) {
	reg := NewRegistry()
	reg.MustRegister(testFeature{name: "base"})

	t.Run("known", func(t *testing.T) {
		_, err := reg.resolve("base")
		if err != nil {
			t.Errorf("resolve: %v", err)
		}
	})
	t.Run("unknown", func(t *testing.T) {
		_, err := reg.resolve("nope")
		if err == nil {
			t.Error("expected error for unknown feature")
		}
	})
}
