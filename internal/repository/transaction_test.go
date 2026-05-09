package repository

import "testing"

type fakeUoW struct{}

func (fakeUoW) Do(_ interface{}, _ func(interface{}) error) error { return nil }

func TestUnitOfWorkShape(t *testing.T) {
	var _ UnitOfWork = nopUnitOfWork{}
}

type nopUnitOfWork struct{}

func (nopUnitOfWork) Do(_ any, fn func(any) error) error {
	if fn != nil {
		return fn(nil)
	}
	return nil
}
