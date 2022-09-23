// Code generated by mockery v2.14.0. DO NOT EDIT.

package buildtest

import (
	mock "github.com/stretchr/testify/mock"
	common "gitlab.com/gitlab-org/gitlab-runner/common"
)

// MockBuildSetupFn is an autogenerated mock type for the BuildSetupFn type
type MockBuildSetupFn struct {
	mock.Mock
}

// Execute provides a mock function with given fields: build
func (_m *MockBuildSetupFn) Execute(build *common.Build) {
	_m.Called(build)
}

type mockConstructorTestingTNewMockBuildSetupFn interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockBuildSetupFn creates a new instance of MockBuildSetupFn. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockBuildSetupFn(t mockConstructorTestingTNewMockBuildSetupFn) *MockBuildSetupFn {
	mock := &MockBuildSetupFn{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
