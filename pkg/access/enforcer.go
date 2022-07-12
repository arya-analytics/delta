package access

type Request struct {
	Subject  Subject
	Resource Resource
	Action   Action
}

type Enforcer interface {
	Enforce(requests ...Request) error
}
