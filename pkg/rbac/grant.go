package rbac

import (
	"github.com/arya-analytics/delta/pkg/access"
	"github.com/arya-analytics/x/gorp"
	"github.com/arya-analytics/x/path"
	"github.com/arya-analytics/x/set"
	"github.com/google/uuid"
)

type GrantRetrieve = gorp.Retrieve[uuid.UUID, Grant]

func WhereGrantSubjectKey(q GrantRetrieve, key access.SubjectKey) GrantRetrieve {
	return q.Where(func(g Grant) bool { return g.SubjectKeys.Contains(key) })
}

type Grant struct {
	Key           uuid.UUID
	SubjectKeys   set.Set[access.SubjectKey]
	ResourcePaths set.Set[path.Path]
	ActionTypes   set.Set[access.ActionType]
}

func (g Grant) GorpKey() uuid.UUID {
	if g.Key == uuid.Nil {
		g.Key = uuid.New()
	}
	return g.Key
}

func (g Grant) SetOptions() []interface{} { return nil }

func (g *Grant) Test(req access.Request) bool {

}
