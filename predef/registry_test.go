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

	array, ok := registry.Types["Array"]
	if !ok {
		t.Fatalf("expected Array descriptor to be loaded")
	}
	if array.Kind != KindInterface {
		t.Fatalf("expected Array to be an interface, got %s", array.Kind)
	}
	if len(array.Methods) != 10 {
		t.Fatalf("expected Array to expose 10 methods, got %#v", array)
	}

	option, ok := registry.Types["Option"]
	if !ok {
		t.Fatalf("expected Option descriptor to be loaded")
	}
	if option.Kind != KindClass {
		t.Fatalf("expected Option to be a class, got %s", option.Kind)
	}

	printer, ok := registry.Types["Printer"]
	if !ok {
		t.Fatalf("expected Printer descriptor to be loaded")
	}
	if printer.Kind != KindInterface || len(printer.Methods) != 3 {
		t.Fatalf("expected Printer to expose 3 methods, got %#v", printer)
	}

	osValue, ok := registry.Types["OS"]
	if !ok {
		t.Fatalf("expected OS descriptor to be loaded")
	}
	if osValue.Kind != KindObject || len(osValue.Fields) != 2 {
		t.Fatalf("expected OS object descriptor, got %#v", osValue)
	}

	str, ok := registry.Types["Str"]
	if !ok {
		t.Fatalf("expected Str descriptor to be loaded")
	}
	if str.Kind != KindInterface {
		t.Fatalf("expected Str to be an interface, got %s", str.Kind)
	}
	if len(str.Methods) != 1 || str.Methods[0].Name != "size" {
		t.Fatalf("expected Str to expose size(), got %#v", str.Methods)
	}
}
