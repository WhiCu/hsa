package invite

type Policy struct {
	maxActive int
	unlimited bool
}

func NewPolicy(maxActive int) Policy { return Policy{maxActive: maxActive} }
func NewUnlimitedPolicy() Policy     { return Policy{unlimited: true} }

func (p Policy) CanIssue(activeCount int) error {
	if p.unlimited {
		return nil
	}
	if activeCount >= p.maxActive {
		return ErrTooManyActive
	}
	return nil
}
