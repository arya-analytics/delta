package ontology

const (
	BuiltIn   Type = "builtin"
	RouteType Type = "route"
)

var Root = ID{Type: BuiltIn, Key: "root"}

func RouteKey(path string) ID {
	return ID{Type: RouteType, Key: path}
}
