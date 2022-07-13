package resource

import (
	"github.com/arya-analytics/x/gorp"
	"github.com/cockroachdb/errors"
)

// DAG implements a key-value backed directed acyclic graph. It represents the central
// data structure for building relationships between resources.
type DAG struct{ db *gorp.DB }

// OpenDAG opens a new DAG whose data is stored in the given database.
func OpenDAG(db *gorp.DB) *DAG { return &DAG{db: db} }

// SetResource sets the given resource in the DAG. Both the key and the resource type
// must be valid or an error will be returned. If the resource already exists, SetResource
// will update the existing resource.
func (d *DAG) SetResource(res Resource) error {
	if err := res.Validate(); err != nil {
		return err
	}
	return gorp.NewCreate[string, Resource]().Entry(&res).Exec(d.db)
}

// GetResource returns the resource with the given key. If the resource does not exist,
// GetResource will return a query.NotFound error.
func (d *DAG) GetResource(key string) (Resource, error) {
	var r Resource
	return r, gorp.NewRetrieve[string, Resource]().WhereKeys(key).Entry(&r).Exec(d.db)
}

// DeleteResource deletes the resource with the given key along with all parent and
// child relationships. If the resource does not exist, DeleteResource will return
// a query.NotFound error.
func (d *DAG) DeleteResource(key string) error {
	if err := d.deleteParentRelationships(key); err != nil {
		return err
	}
	if err := d.deleteChildRelationships(key); err != nil {
		return err
	}
	return d.deleteResource(key)
}

// SetRelationship sets the given relationship between two resources in the DAG. Both
// the parent and child resources must exist. If the relationship already exists,
// SetRelationship will update the existing relationship.
func (d *DAG) SetRelationship(rel Relationship) error {
	if _, err := d.GetResource(rel.Parent); err != nil {
		return errors.Wrapf(
			err,
			"[resource] - parent resource %s does not exist",
		)
	}
	if _, err := d.GetResource(rel.Child); err != nil {
		return errors.Wrapf(
			err, "[resource] - parent resource %s does not exist", rel.Child)
	}
	descendants, err := d.getDescendants(rel.Child)
	if err != nil {
		return err
	}
	if _, exists := descendants[rel.Parent]; exists {
		return errors.New("[resource] - cyclic violation")
	}
	return d.setRelationship(rel)
}

// DeleteRelationship deletes the relationship between two resources in the DAG.
// Returns an error if the relationship does not exist.
func (d *DAG) DeleteRelationship(rel Relationship) error {
	return d.deleteRelationship(rel)
}

// GetParentResources returns the resources that are parents of the given resource.
// If the resource does not exist, GetParentResources will return a query.NotFound error.
// If the resource has no parents, GetParentResources will return an empty slice.
func (d *DAG) GetParentResources(key string) ([]Resource, error) {
	relationships, err := d.getRelationships(func(rel Relationship) bool {
		return rel.Child == key
	})
	if err != nil {
		return nil, err
	}
	var keys []string
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
func (d *DAG) GetChildResources(key string) ([]Resource, error) {
	relationships, err := d.getRelationships(func(rel Relationship) bool {
		return rel.Parent == key
	})
	if err != nil {
		return nil, err
	}
	var keys []string
	for _, rel := range relationships {
		keys = append(keys, rel.Child)
	}
	return d.getResources(keys)
}

func (d *DAG) getResources(keys []string) ([]Resource, error) {
	var resources []Resource
	return resources, gorp.NewRetrieve[string, Resource]().WhereKeys(keys...).Entries(&resources).Exec(d.db)
}

func (d *DAG) getRelationships(matcher func(Relationship) bool) ([]Relationship, error) {
	var relationships []Relationship
	return relationships, gorp.NewRetrieve[string, Relationship]().Where(matcher).Entries(&relationships).Exec(d.db)
}

func (d *DAG) getAncestors(key string) (map[string]Resource, error) {
	ancestors := make(map[string]Resource)
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

func (d *DAG) getDescendants(key string) (map[string]Resource, error) {
	descendants := make(map[string]Resource)
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

func (d *DAG) deleteResource(key string) error {
	return gorp.NewDelete[string, Resource]().WhereKeys(key).Exec(d.db)
}

func (d *DAG) deleteParentRelationships(key string) error {
	return gorp.NewDelete[string, Relationship]().Where(func(rel Relationship) bool {
		return rel.Child == key
	}).Exec(d.db)
}

func (d *DAG) deleteChildRelationships(key string) error {
	return gorp.NewDelete[string, Relationship]().Where(func(rel Relationship) bool {
		return rel.Parent == key
	}).Exec(d.db)
}

func (d *DAG) setRelationship(rel Relationship) error {
	return gorp.NewCreate[string, Relationship]().Entry(&rel).Exec(d.db)
}

func (d *DAG) deleteRelationship(rel Relationship) error {
	return gorp.NewDelete[string, Relationship]().Where(func(rel Relationship) bool {
		return rel.Parent == rel.Child
	}).Exec(d.db)
}
