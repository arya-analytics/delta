package resource

import (
	"fmt"
)

type Type string

type Resource struct {
	Key  string
	Type Type
}

func (r Resource) GorpKey() string {
	return fmt.Sprintf(r.Key)
}

func (r Resource) SetOptions() []interface{} { return nil }

func (r Resource) Validate() error {
	if r.Key == "" {
		return fmt.Errorf("[resource] - key is required")
	}
	if r.Type == "" {
		return fmt.Errorf("[resource] - type is required")
	}
	return nil
}
