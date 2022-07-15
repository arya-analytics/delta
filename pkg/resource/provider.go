package resource

type Provider interface {
	GetAttributes(key string) (Attributes, error)
}
