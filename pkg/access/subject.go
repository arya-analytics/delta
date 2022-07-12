package access

import (
	"github.com/arya-analytics/x/path"
	"github.com/arya-analytics/x/set"
)

type SubjectKey string

// Subject represents an entity that executes an Action on a Resource. A Subject
//
type Subject struct {
	Key     SubjectKey
	Parents set.Set[path.Path]
}
