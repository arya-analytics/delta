package resource

import (
	"github.com/arya-analytics/x/gorp"
	"github.com/cockroachdb/errors"
)

// DAG is a key-value backed directed acyclic graph that implements the Writer
// interface. It represents the central data structure for building relationships
// between resources.
type DAG struct{ Txn gorp.Txn }

// DefineResource defines the given resource in the DAG. Both the key and the resource type
// must be valid or an error will be returned. If the resource already exists,
// DefineResource will do nothing.
func (d DAG) DefineResource(tk Key) error {
	if err := tk.Validate(); err != nil {
		return err
	}
	return gorp.NewCreate[Key, Resource]().
		Entry(&Resource{Key: tk}).
		Exec(d.Txn)
}

// GetResource returns the resource with the given key. If the resource does not exist,
// GetResource will return a query.NotFound error.
func (d DAG) GetResource(tk Key) (Resource, error) {
	var r Resource
	return r, gorp.NewRetrieve[Key, Resource]().
		WhereKeys(tk).
		Entry(&r).
		Exec(d.Txn)
}

// DeleteResource deletes the resource with the given key along with all parent and
// child relationships. If the resource does not exist, DeleteResource will return
// a query.NotFound error.
func (d DAG) DeleteResource(tk Key) error {
	if err := d.deleteParentRelationships(tk); err != nil {
		return err
	}
	if err := d.deleteChildRelationships(tk); err != nil {
		return err
	}
	return d.deleteResource(tk)
}

// DefineRelationship defines a relationship between two resources in the DAG.
// Both the parent and child resources must exist. If the relationship already exists,
// SetRelationship will update the existing relationship.
func (d DAG) DefineRelationship(child, parent Key) error {
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
func (d DAG) DeleteRelationship(parent, child Key) error {
	return d.deleteRelationship(Relationship{Parent: parent, Child: child})
}

// GetParentResources returns the resources that are parents of the given resource.
// If the resource does not exist, GetParentResources will return a query.NotFound error.
// If the resource has no parents, GetParentResources will return an empty slice.
func (d DAG) GetParentResources(key Key) ([]Resource, error) {
	relationships, err := d.getRelationships(func(rel Relationship) bool {
		return rel.Child == key
	})
	if err != nil {
		return nil, err
	}
	var keys []Key
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
func (d DAG) GetChildResources(key Key) ([]Resource, error) {
	relationships, err := d.getRelationships(func(rel Relationship) bool {
		return rel.Parent == key
	})
	if err != nil {
		return nil, err
	}
	var keys []Key
	for _, rel := range relationships {
		keys = append(keys, rel.Child)
	}
	return d.getResources(keys)
}

func (d DAG) getResources(keys []Key) ([]Resource, error) {
	var resources []Resource
	return resources, gorp.NewRetrieve[Key, Resource]().
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

func (d DAG) getAncestors(key Key) (map[Key]Resource, error) {
	ancestors := make(map[Key]Resource)
	parents, err := d.GetParentResources(key)
	if err != nil {
		return nil, err
	}
	for _, parent := range parents {
		parentAncestors, err := d.getAncestors(parent.Key)
		if err != nil {
			return nil, err
		}
		for k, v := range parentAncestors {
			ancestors[k] = v
		}
		ancestors[parent.Key] = parent
	}
	return ancestors, nil
}

func (d DAG) getDescendants(key Key) (map[Key]Resource, error) {
	descendants := make(map[Key]Resource)
	children, err := d.GetChildResources(key)
	if err != nil {
		return nil, err
	}
	if len(children) == 0 {
		return nil, nil
	}
	for _, child := range children {
		childDescendants, err := d.getDescendants(child.Key)
		if err != nil {
			return nil, err
		}
		for k, v := range childDescendants {
			descendants[k] = v
		}
		descendants[child.Key] = child
	}
	return descendants, nil
}

func (d DAG) deleteResource(tk Key) error {
	return gorp.NewDelete[Key, Resource]().
		WhereKeys(tk).
		Exec(d.Txn)
}

func (d DAG) deleteParentRelationships(tk Key) error {
	return gorp.NewDelete[string, Relationship]().Where(func(rel Relationship) bool {
		return rel.Child == tk
	}).Exec(d.Txn)
}

func (d DAG) deleteChildRelationships(tk Key) error {
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
