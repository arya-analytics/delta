package iterator

import "github.com/arya-analytics/x/confluence"

type requestConfluence struct{ confluence.Confluence[Request] }

func newRequestConfluence() requestSegment { return &requestConfluence{} }
