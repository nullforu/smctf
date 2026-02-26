package stack

import (
	"encoding/json"
	"testing"
)

func TestTargetPortSpecsValueAndScan(t *testing.T) {
	specs := TargetPortSpecs{{ContainerPort: 80, Protocol: "TCP"}}
	value, err := specs.Value()
	if err != nil {
		t.Fatalf("Value error: %v", err)
	}

	raw, ok := value.(string)
	if !ok {
		t.Fatalf("expected string value, got %T", value)
	}

	var decoded []TargetPortSpec
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}

	if len(decoded) != 1 || decoded[0].ContainerPort != 80 || decoded[0].Protocol != "TCP" {
		t.Fatalf("unexpected decoded value: %+v", decoded)
	}

	var scanned TargetPortSpecs
	if err := scanned.Scan([]byte(raw)); err != nil {
		t.Fatalf("Scan []byte error: %v", err)
	}

	if len(scanned) != 1 || scanned[0].Protocol != "TCP" {
		t.Fatalf("unexpected scanned value: %+v", scanned)
	}

	var scannedString TargetPortSpecs
	if err := scannedString.Scan(raw); err != nil {
		t.Fatalf("Scan string error: %v", err)
	}

	if len(scannedString) != 1 || scannedString[0].ContainerPort != 80 {
		t.Fatalf("unexpected scanned string value: %+v", scannedString)
	}

	var scannedNil TargetPortSpecs
	if err := scannedNil.Scan(nil); err != nil {
		t.Fatalf("Scan nil error: %v", err)
	}

	if scannedNil != nil {
		t.Fatalf("expected nil after scanning nil, got %+v", scannedNil)
	}

	if err := scannedNil.Scan(123); err == nil {
		t.Fatalf("expected error for unsupported Scan type")
	}
}

func TestPortMappingsValueAndScan(t *testing.T) {
	mappings := PortMappings{{ContainerPort: 80, Protocol: "TCP", NodePort: 31001}}
	value, err := mappings.Value()
	if err != nil {
		t.Fatalf("Value error: %v", err)
	}

	raw, ok := value.(string)
	if !ok {
		t.Fatalf("expected string value, got %T", value)
	}

	var decoded []PortMapping
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}

	if len(decoded) != 1 || decoded[0].NodePort != 31001 {
		t.Fatalf("unexpected decoded value: %+v", decoded)
	}

	var scanned PortMappings
	if err := scanned.Scan([]byte(raw)); err != nil {
		t.Fatalf("Scan []byte error: %v", err)
	}

	if len(scanned) != 1 || scanned[0].Protocol != "TCP" {
		t.Fatalf("unexpected scanned value: %+v", scanned)
	}

	var scannedString PortMappings
	if err := scannedString.Scan(raw); err != nil {
		t.Fatalf("Scan string error: %v", err)
	}

	if len(scannedString) != 1 || scannedString[0].ContainerPort != 80 {
		t.Fatalf("unexpected scanned string value: %+v", scannedString)
	}

	var scannedNil PortMappings
	if err := scannedNil.Scan(nil); err != nil {
		t.Fatalf("Scan nil error: %v", err)
	}

	if scannedNil != nil {
		t.Fatalf("expected nil after scanning nil, got %+v", scannedNil)
	}

	if err := scannedNil.Scan(123); err == nil {
		t.Fatalf("expected error for unsupported Scan type")
	}
}
