package invite

type Policy struct {
	maxActive int
}

func NewPolicy(maxActive int) Policy {
	return Policy{maxActive: maxActive}
}

func (p Policy) CanIssue(activeCount int) error {
	if activeCount >= p.maxActive {
		return ErrTooManyActive
	}
	return nil
}
