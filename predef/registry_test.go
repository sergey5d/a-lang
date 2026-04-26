package predef

import "testing"

func TestLoadRegistry(t *testing.T) {
	registry, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	list, ok := registry.Types["List"]
	if !ok {
		t.Fatalf("expected List descriptor to be loaded")
	}
	if list.Kind != KindInterface {
		t.Fatalf("expected List to be an interface, got %s", list.Kind)
	}
	if len(list.Methods) == 0 {
		t.Fatalf("expected List to declare methods")
	}

	option, ok := registry.Types["Option"]
	if !ok {
		t.Fatalf("expected Option descriptor to be loaded")
	}
	if option.Kind != KindClass {
		t.Fatalf("expected Option to be a class, got %s", option.Kind)
	}

	term, ok := registry.Types["Term"]
	if !ok {
		t.Fatalf("expected Term descriptor to be loaded")
	}
	if len(term.Methods) != 2 {
		t.Fatalf("expected Term to expose 2 methods, got %d", len(term.Methods))
	}
}
