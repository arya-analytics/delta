package resource

type attributeReader struct {
	Providers providers
	dag       DAG
}

func (r attributeReader) GetResource(key TypeKey) (Resource, error) {
	res, err := r.dag.GetResource(key)
	if err != nil {
		return res, err
	}
	res.Attrs, err = r.Providers.GetAttributes(r.dag.Txn, key)
	return res, err
}

func (r attributeReader) GetChildResources(key TypeKey) ([]Resource, error) {
	children, err := r.dag.GetChildResources(key)
	if err != nil {
		return nil, err
	}
	return r.getAttributes(children)
}

func (r attributeReader) GetParentResources(key TypeKey) ([]Resource, error) {
	parents, err := r.dag.GetParentResources(key)
	if err != nil {
		return nil, err
	}
	return r.getAttributes(parents)
}

func (r attributeReader) getAttributes(resources []Resource) ([]Resource, error) {
	var err error
	for i, res := range resources {
		resources[i].Attrs, err = r.Providers.GetAttributes(r.dag.Txn, res.TypeKey)
	}
	return resources, err
}
