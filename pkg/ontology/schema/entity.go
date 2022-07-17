package schema

type Entity struct {
	schema *Schema
	data   map[string]interface{}
}

func Get[V Value](d Entity, k string) (V, bool) {
	v, ok := d.data[k]
	return v, ok
}

func Set[V Value](D Entity, k string, v V) {
	f, ok := D.schema.Fields[k]
	if !ok {
		panic("[schema] - field not found")
	}
	if !f.Type.AssertValue(v) {
		panic("[schema] - invalid field type")
	}
	D.data[k] = v
}
