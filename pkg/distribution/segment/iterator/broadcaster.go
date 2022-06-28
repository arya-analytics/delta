package iterator

import "github.com/arya-analytics/x/confluence"

type requestBroadcaster struct{ confluence.Confluence[Request] }
