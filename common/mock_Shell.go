// Code generated by mockery v2.14.0. DO NOT EDIT.

package common

import mock "github.com/stretchr/testify/mock"

// MockShell is an autogenerated mock type for the Shell type
type MockShell struct {
	mock.Mock
}

// GenerateScript provides a mock function with given fields: buildStage, info
func (_m *MockShell) GenerateScript(buildStage BuildStage, info ShellScriptInfo) (string, error) {
	ret := _m.Called(buildStage, info)

	var r0 string
	if rf, ok := ret.Get(0).(func(BuildStage, ShellScriptInfo) string); ok {
		r0 = rf(buildStage, info)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(BuildStage, ShellScriptInfo) error); ok {
		r1 = rf(buildStage, info)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetConfiguration provides a mock function with given fields: info
func (_m *MockShell) GetConfiguration(info ShellScriptInfo) (*ShellConfiguration, error) {
	ret := _m.Called(info)

	var r0 *ShellConfiguration
	if rf, ok := ret.Get(0).(func(ShellScriptInfo) *ShellConfiguration); ok {
		r0 = rf(info)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*ShellConfiguration)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(ShellScriptInfo) error); ok {
		r1 = rf(info)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetFeatures provides a mock function with given fields: features
func (_m *MockShell) GetFeatures(features *FeaturesInfo) {
	_m.Called(features)
}

// GetName provides a mock function with given fields:
func (_m *MockShell) GetName() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// IsDefault provides a mock function with given fields:
func (_m *MockShell) IsDefault() bool {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

type mockConstructorTestingTNewMockShell interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockShell creates a new instance of MockShell. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockShell(t mockConstructorTestingTNewMockShell) *MockShell {
	mock := &MockShell{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
