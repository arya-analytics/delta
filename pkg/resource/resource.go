package resource

import (
	"fmt"
)

type Type string

type TypeKey struct {
	Key  string
	Type Type
}

func (tk TypeKey) Validate() error {
	if tk.Key == "" {
		return fmt.Errorf("[resource] - key is required")
	}
	if tk.Type == "" {
		return fmt.Errorf("[resource] - type is required")
	}
	return nil
}

func (tk TypeKey) String() string {
	return fmt.Sprintf("%s:%s", tk.Key, tk.Type)
}

type Attributes struct {
	Name  string
	Extra map[string]interface{}
}

type Resource struct {
	TypeKey
	Attrs Attributes
}

func (r Resource) GorpKey() TypeKey {
	return r.TypeKey
}

func (r Resource) SetOptions() []interface{} { return nil }
