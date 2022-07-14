package resource

import (
	"github.com/arya-analytics/x/gorp"
	"github.com/cockroachdb/errors"
)

// DAG is a key-value backed directed acyclic graph that implements the Writer
// interface. It represents the central data structure for building relationships
// between resources.
type DAG struct{ Txn gorp.Txn }

// SetResource sets the given resource in the DAG. Both the key and the resource type
// must be valid or an error will be returned. If the resource already exists, SetResource
// will update the existing resource.
func (d DAG) SetResource(tk TypeKey) error {
	if err := tk.Validate(); err != nil {
		return err
	}
	return gorp.NewCreate[TypeKey, Resource]().
		Entry(&Resource{TypeKey: tk}).
		Exec(d.Txn)
}

// GetResource returns the resource with the given key. If the resource does not exist,
// GetResource will return a query.NotFound error.
func (d DAG) GetResource(tk TypeKey) (Resource, error) {
	var r Resource
	return r, gorp.NewRetrieve[TypeKey, Resource]().
		WhereKeys(tk).
		Entry(&r).
		Exec(d.Txn)
}

// DeleteResource deletes the resource with the given key along with all parent and
// child relationships. If the resource does not exist, DeleteResource will return
// a query.NotFound error.
func (d DAG) DeleteResource(tk TypeKey) error {
	if err := d.deleteParentRelationships(tk); err != nil {
		return err
	}
	if err := d.deleteChildRelationships(tk); err != nil {
		return err
	}
	return d.deleteResource(tk)
}

// SetRelationship sets the given relationship between two resources in the DAG. Both
// the parent and child resources must exist. If the relationship already exists,
// SetRelationship will update the existing relationship.
func (d DAG) SetRelationship(child, parent TypeKey) error {
	if _, err := d.GetResource(parent); err != nil {
		return errors.Wrapf(
			err,
			"[resource] - parent resource %s does not exist",
		)
	}
	if _, err := d.GetResource(child); err != nil {
		return errors.Wrapf(
			err, "[resource] - parent resource %s does not exist", child)
	}
	descendants, err := d.getDescendants(child)
	if err != nil {
		return err
	}
	if _, exists := descendants[parent]; exists {
		return errors.New("[resource] - cyclic violation")
	}
	return d.setRelationship(Relationship{Parent: parent, Child: child})

}

// DeleteRelationship deletes the relationship between two resources in the DAG.
// Returns an error if the relationship does not exist.
func (d DAG) DeleteRelationship(parent, child TypeKey) error {
	return d.deleteRelationship(Relationship{Parent: parent, Child: child})
}

// GetParentResources returns the resources that are parents of the given resource.
// If the resource does not exist, GetParentResources will return a query.NotFound error.
// If the resource has no parents, GetParentResources will return an empty slice.
func (d DAG) GetParentResources(key TypeKey) ([]Resource, error) {
	relationships, err := d.getRelationships(func(rel Relationship) bool {
		return rel.Child == key
	})
	if err != nil {
		return nil, err
	}
	var keys []TypeKey
	for _, rel := range relationships {
		keys = append(keys, rel.Parent)
	}
	res, err := d.getResources(keys)
	if err != nil {
		panic(err)
	}
	return res, nil
}

// GetChildResources returns the resources that are children of the given resource.
// If the resource does not exist, GetChildResources will return a query.NotFound error.
// If the resource has no children, GetChildResources will return an empty slice.
func (d DAG) GetChildResources(key TypeKey) ([]Resource, error) {
	relationships, err := d.getRelationships(func(rel Relationship) bool {
		return rel.Parent == key
	})
	if err != nil {
		return nil, err
	}
	var keys []TypeKey
	for _, rel := range relationships {
		keys = append(keys, rel.Child)
	}
	return d.getResources(keys)
}

func (d DAG) getResources(keys []TypeKey) ([]Resource, error) {
	var resources []Resource
	return resources, gorp.NewRetrieve[TypeKey, Resource]().
		WhereKeys(keys...).
		Entries(&resources).
		Exec(d.Txn)
}

func (d DAG) getRelationships(matcher func(Relationship) bool) ([]Relationship, error) {
	var relationships []Relationship
	return relationships, gorp.NewRetrieve[string, Relationship]().
		Where(matcher).
		Entries(&relationships).
		Exec(d.Txn)
}

func (d DAG) getAncestors(key TypeKey) (map[TypeKey]Resource, error) {
	ancestors := make(map[TypeKey]Resource)
	parents, err := d.GetParentResources(key)
	if err != nil {
		return nil, err
	}
	for _, parent := range parents {
		parentAncestors, err := d.getAncestors(parent.TypeKey)
		if err != nil {
			return nil, err
		}
		for k, v := range parentAncestors {
			ancestors[k] = v
		}
		ancestors[parent.TypeKey] = parent
	}
	return ancestors, nil
}

func (d DAG) getDescendants(key TypeKey) (map[TypeKey]Resource, error) {
	descendants := make(map[TypeKey]Resource)
	children, err := d.GetChildResources(key)
	if err != nil {
		return nil, err
	}
	if len(children) == 0 {
		return nil, nil
	}
	for _, child := range children {
		childDescendants, err := d.getDescendants(child.TypeKey)
		if err != nil {
			return nil, err
		}
		for k, v := range childDescendants {
			descendants[k] = v
		}
		descendants[child.TypeKey] = child
	}
	return descendants, nil
}

func (d DAG) deleteResource(tk TypeKey) error {
	return gorp.NewDelete[TypeKey, Resource]().
		WhereKeys(tk).
		Exec(d.Txn)
}

func (d DAG) deleteParentRelationships(tk TypeKey) error {
	return gorp.NewDelete[string, Relationship]().Where(func(rel Relationship) bool {
		return rel.Child == tk
	}).Exec(d.Txn)
}

func (d DAG) deleteChildRelationships(tk TypeKey) error {
	return gorp.NewDelete[string, Relationship]().Where(func(rel Relationship) bool {
		return rel.Parent == tk
	}).Exec(d.Txn)
}

func (d DAG) setRelationship(rel Relationship) error {
	return gorp.NewCreate[string, Relationship]().Entry(&rel).Exec(d.Txn)
}

func (d DAG) deleteRelationship(rel Relationship) error {
	return gorp.NewDelete[string, Relationship]().WhereKeys(rel.GorpKey()).Exec(d.Txn)
}
