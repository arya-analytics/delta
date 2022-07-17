package ontology

const (
	BuiltIn   Type = "builtin"
	RouteType Type = "route"
)

var RootKey = Key{Type: BuiltIn, Key: "root"}

func RouteKey(path string) Key {
	return Key{Type: RouteType, Key: path}
}
