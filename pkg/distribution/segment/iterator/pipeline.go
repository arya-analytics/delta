package iterator

import "github.com/arya-analytics/x/confluence"

const (
	acknowledgeAddr = "acknowledge"
	dataAddr        = "data"
)

type (
	requestSegment  = confluence.Segment[Request]
	responseSegment = confluence.Segment[Response]
)
