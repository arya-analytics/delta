package ontology

type Service interface {
	Schema() *Schema
	Retrieve(key string) (Entity, error)
}

type services map[Type]Service

func (s services) Register(svc Service) {
	t := svc.Schema().Type
	if _, ok := s[t]; ok {
		panic("[ontology] - service already registered")
	}
	s[t] = svc
}

func (s services) Retrieve(key ID) (Entity, error) {
	svc, ok := s[key.Type]
	if !ok {
		panic("[ontology] - service not found")
	}
	return svc.Retrieve(key.Key)
}
