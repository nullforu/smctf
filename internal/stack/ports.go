package stack

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type TargetPortSpecs []TargetPortSpec

func (p TargetPortSpecs) Value() (driver.Value, error) {
	if p == nil {
		return nil, nil
	}

	payload, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("stack target ports marshal: %w", err)
	}

	return string(payload), nil
}

func (p *TargetPortSpecs) Scan(value any) error {
	if value == nil {
		*p = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, p)
	case string:
		return json.Unmarshal([]byte(v), p)
	default:
		return fmt.Errorf("stack target ports scan: %T", value)
	}
}

type PortMappings []PortMapping

func (p PortMappings) Value() (driver.Value, error) {
	if p == nil {
		return nil, nil
	}

	payload, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("stack port mappings marshal: %w", err)
	}

	return string(payload), nil
}

func (p *PortMappings) Scan(value any) error {
	if value == nil {
		*p = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, p)
	case string:
		return json.Unmarshal([]byte(v), p)
	default:
		return fmt.Errorf("stack port mappings scan: %T", value)
	}
}
