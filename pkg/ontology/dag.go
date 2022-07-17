package ontology

import (
	"github.com/arya-analytics/x/gorp"
	"github.com/arya-analytics/x/query"
	"github.com/cockroachdb/errors"
)

// DAG is a key-value backed directed acyclic graph that implements the Writer
// interface. It represents the central data structure for building relationships
// between resources.
type DAG struct{ DB gorp.Txn }

// DefineResource defines the given resource in the DAG. Both the key and the resource type
// must be valid or an error will be returned. If the resource already exists,
// DefineResource will do nothing.
func (d DAG) DefineResource(tk ID) error {
	if err := tk.Validate(); err != nil {
		return err
	}
	return gorp.NewCreate[ID, Resource]().
		Entry(&Resource{Key: tk}).
		Exec(d.DB)
}

// GetResource returns the resource with the given key. If the resource does not exist,
// GetResource will return a query.NotFound error.
func (d DAG) RetrieveResource(tk ID) (Resource, error) {
	var r Resource
	return r, gorp.NewRetrieve[ID, Resource]().
		WhereKeys(tk).
		Entry(&r).
		Exec(d.DB)
}

// DeleteResource deletes the resource with the given key along with all parent and
// child relationships. If the resource does not exist, DeleteResource will return
// a query.NotFound error.
func (d DAG) DeleteResource(tk ID) error {
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
func (d DAG) DefineRelationship(from, to ID, t RelationshipType) error {
	rel := Relationship{From: from, To: to, Type: t}
	exists, err := d.checkRelationshipExists(rel)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	if _, err := d.RetrieveResource(to); err != nil {
		return errors.Wrapf(
			err,
			"[resource] - to resource %s does not exist",
		)
	}
	if _, err := d.RetrieveResource(from); err != nil {
		return errors.Wrapf(
			err, "[resource] - to resource %s does not exist", from)
	}
	descendants, err := d.getDescendants(from)
	if err != nil {
		return err
	}
	if _, exists := descendants[to]; exists {
		return errors.New("[resource] - cyclic violation")
	}
	return d.setRelationship(rel)

}

// DeleteRelationship deletes the relationship between two resources in the DAG.
// Returns an error if the relationship does not exist.
func (d DAG) DeleteRelationship(parent, child ID, t RelationshipType) error {
	return d.deleteRelationship(Relationship{From: parent, To: child, Type: t})
}

// GetParentResources returns the resources that are parents of the given resource.
// If the resource does not exist, GetParentResources will return a query.NotFound error.
// If the resource has no parents, GetParentResources will return an empty slice.
func (d DAG) RetrieveParentResources(key ID) ([]Resource, error) {
	relationships, err := d.getRelationships(func(rel *Relationship) bool {
		return rel.To == key
	})
	if err != nil {
		return nil, err
	}
	var keys []ID
	for _, rel := range relationships {
		keys = append(keys, rel.From)
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
func (d DAG) RetrieveChildResources(key ID) ([]Resource, error) {
	relationships, err := d.getRelationships(func(rel *Relationship) bool {
		return rel.From == key
	})
	if err != nil {
		return nil, err
	}
	var keys []ID
	for _, rel := range relationships {
		keys = append(keys, rel.To)
	}
	return d.getResources(keys)
}

func (d DAG) IterParents(key ID) func() ([]Resource, error) {
	nextKeys := []ID{key}
	return func() ([]Resource, error) {
		var resources []Resource
		for _, k := range nextKeys {
			pr, err := d.RetrieveParentResources(k)
			if err != nil && err != query.NotFound {
				return nil, err
			}
			resources = append(resources, pr...)
		}
		if len(resources) == 0 {
			return nil, query.NotFound
		}
		for _, res := range resources {
			nextKeys = append(nextKeys, res.Key)
		}
		return resources, nil
	}
}

func (d DAG) getResources(keys []ID) ([]Resource, error) {
	var resources []Resource
	return resources, gorp.NewRetrieve[ID, Resource]().
		WhereKeys(keys...).
		Entries(&resources).
		Exec(d.DB)
}

func (d DAG) getRelationships(matcher func(*Relationship) bool) ([]Relationship, error) {
	var relationships []Relationship
	return relationships, gorp.NewRetrieve[string, Relationship]().
		Where(matcher).
		Entries(&relationships).
		Exec(d.DB)
}

func (d DAG) getAncestors(key ID) (map[ID]Resource, error) {
	ancestors := make(map[ID]Resource)
	parents, err := d.RetrieveParentResources(key)
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

func (d DAG) getDescendants(key ID) (map[ID]Resource, error) {
	descendants := make(map[ID]Resource)
	children, err := d.RetrieveChildResources(key)
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

func (d DAG) deleteResource(tk ID) error {
	return gorp.NewDelete[ID, Resource]().
		WhereKeys(tk).
		Exec(d.DB)
}

func (d DAG) deleteParentRelationships(tk ID) error {
	return gorp.NewDelete[string, Relationship]().Where(func(rel *Relationship) bool {
		return rel.To == tk
	}).Exec(d.DB)
}

func (d DAG) deleteChildRelationships(tk ID) error {
	return gorp.NewDelete[string, Relationship]().Where(func(rel *Relationship) bool {
		return rel.From == tk
	}).Exec(d.DB)
}

func (d DAG) setRelationship(rel Relationship) error {
	return gorp.NewCreate[string, Relationship]().Entry(&rel).Exec(d.DB)
}

func (d DAG) deleteRelationship(rel Relationship) error {
	return gorp.NewDelete[string, Relationship]().WhereKeys(rel.GorpKey()).Exec(d.DB)
}

func (d DAG) checkRelationshipExists(rel Relationship) (bool, error) {
	return gorp.NewRetrieve[string, Relationship]().WhereKeys(rel.GorpKey()).Exists(d.DB)
}
