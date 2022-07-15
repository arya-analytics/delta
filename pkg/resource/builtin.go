package resource

const (
	RouteType Type = "route"
)

func RouteKey(path string) Key {
	return Key{Type: RouteType, Key: path}
}
