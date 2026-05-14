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
	if len(array.Methods) != 15 {
		t.Fatalf("expected Array to expose 15 methods, got %#v", array)
	}

	option, ok := registry.Types["Option"]
	if !ok {
		t.Fatalf("expected Option descriptor to be loaded")
	}
	if option.Kind != KindEnum {
		t.Fatalf("expected Option to be an enum, got %s", option.Kind)
	}

	tuple2, ok := registry.Types["Tuple2"]
	if !ok {
		t.Fatalf("expected Tuple2 descriptor to be loaded")
	}
	if tuple2.Kind != KindRecord {
		t.Fatalf("expected Tuple2 to be a record, got %s", tuple2.Kind)
	}
	if len(tuple2.Fields) != 2 {
		t.Fatalf("expected Tuple2 to expose 2 fields, got %#v", tuple2.Fields)
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
	if len(str.Methods) != 3 {
		t.Fatalf("expected Str to expose 3 methods, got %#v", str.Methods)
	}
	if str.Methods[0].Name != "size" || str.Methods[1].Name != "split" || str.Methods[2].Name != "indexOf" {
		t.Fatalf("unexpected Str methods %#v", str.Methods)
	}
}
