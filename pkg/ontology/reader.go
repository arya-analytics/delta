package ontology

type attributeReader struct {
	Providers services
	dag       DAG
}

func (r attributeReader) RetrieveResource(key Key) (Resource, error) {
	res, err := r.dag.RetrieveResource(key)
	if err != nil {
		return res, err
	}
	res.Data, err = r.Providers.Retrieve(key)
	return res, err
}

func (r attributeReader) RetrieveChildResources(key Key) ([]Resource, error) {
	children, err := r.dag.RetrieveChildResources(key)
	if err != nil {
		return nil, err
	}
	return r.getAttributes(children)
}

func (r attributeReader) RetrieveParentResources(key Key) ([]Resource, error) {
	parents, err := r.dag.RetrieveParentResources(key)
	if err != nil {
		return nil, err
	}
	return r.getAttributes(parents)
}

func (r attributeReader) getAttributes(resources []Resource) ([]Resource, error) {
	var err error
	for i, res := range resources {
		resources[i].Data, err = r.Providers.Retrieve(res.Key)
	}
	return resources, err
}
