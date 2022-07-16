package ontology

type Provider interface {
	GetAttributes(key string) (Attributes, error)
}
