package resource

import "fmt"

type Relationship struct {
	Parent string
	Child  string
}

func (r Relationship) GorpKey() string {
	return fmt.Sprintf("%s-%s", r.Parent, r.Child)
}

func (r Relationship) SetOptions() []interface{} { return nil }

func (r Relationship) Validate() error {
	if r.Parent == "" {
		return fmt.Errorf("[resource] - relationship parent is required")
	}
	if r.Child == "" {
		return fmt.Errorf("[resource] - relationship child is required")
	}
	if r.Parent == r.Child {
		return fmt.Errorf("[resource] - relationship parent and child are the same")
	}
	return nil
}
