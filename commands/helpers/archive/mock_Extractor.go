// Code generated by mockery v2.14.0. DO NOT EDIT.

package archive

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// MockExtractor is an autogenerated mock type for the Extractor type
type MockExtractor struct {
	mock.Mock
}

// Extract provides a mock function with given fields: ctx
func (_m *MockExtractor) Extract(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewMockExtractor interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockExtractor creates a new instance of MockExtractor. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockExtractor(t mockConstructorTestingTNewMockExtractor) *MockExtractor {
	mock := &MockExtractor{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
