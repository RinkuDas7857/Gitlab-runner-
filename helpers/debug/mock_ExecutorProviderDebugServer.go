// Code generated by mockery v1.1.0. DO NOT EDIT.

package debug

import (
	mux "github.com/gorilla/mux"
	mock "github.com/stretchr/testify/mock"
)

// MockExecutorProviderDebugServer is an autogenerated mock type for the ExecutorProviderDebugServer type
type MockExecutorProviderDebugServer struct {
	mock.Mock
}

// ServeDebugHTTP provides a mock function with given fields: router
func (_m *MockExecutorProviderDebugServer) ServeDebugHTTP(router *mux.Router) {
	_m.Called(router)
}
