package ontology

import "fmt"

type Relationship struct {
	Parent Key
	Child  Key
}

func (r Relationship) GorpKey() string {
	return fmt.Sprintf("%s:%s", r.Parent.String(), r.Child.String())

}

func (r Relationship) SetOptions() []interface{} { return nil }

func (r Relationship) Validate() error {
	if r.Parent.Key == "" {
		return fmt.Errorf("[resource] - relationship parent is required")
	}
	if r.Child.Key == "" {
		return fmt.Errorf("[resource] - relationship child is required")
	}
	if r.Parent == r.Child {
		return fmt.Errorf("[resource] - relationship parent and child are the same")
	}
	return nil
}
