// Code generated by mockery v1.0.0. DO NOT EDIT.

package parser

import mock "github.com/stretchr/testify/mock"

// MockParser is an autogenerated mock type for the Parser type
type MockParser struct {
	mock.Mock
}

// ParseVolume provides a mock function with given fields: spec
func (_m *MockParser) ParseVolume(spec string) (*Volume, error) {
	ret := _m.Called(spec)

	var r0 *Volume
	if rf, ok := ret.Get(0).(func(string) *Volume); ok {
		r0 = rf(spec)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*Volume)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(spec)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
