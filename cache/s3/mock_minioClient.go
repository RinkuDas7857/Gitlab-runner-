// Code generated by mockery v2.14.0. DO NOT EDIT.

package s3

import (
	context "context"
	http "net/http"

	mock "github.com/stretchr/testify/mock"

	time "time"

	url "net/url"
)

// mockMinioClient is an autogenerated mock type for the minioClient type
type mockMinioClient struct {
	mock.Mock
}

// PresignHeader provides a mock function with given fields: ctx, method, bucketName, objectName, expires, reqParams, extraHeaders
func (_m *mockMinioClient) PresignHeader(ctx context.Context, method string, bucketName string, objectName string, expires time.Duration, reqParams url.Values, extraHeaders http.Header) (*url.URL, error) {
	ret := _m.Called(ctx, method, bucketName, objectName, expires, reqParams, extraHeaders)

	var r0 *url.URL
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, time.Duration, url.Values, http.Header) *url.URL); ok {
		r0 = rf(ctx, method, bucketName, objectName, expires, reqParams, extraHeaders)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*url.URL)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string, string, time.Duration, url.Values, http.Header) error); ok {
		r1 = rf(ctx, method, bucketName, objectName, expires, reqParams, extraHeaders)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTnewMockMinioClient interface {
	mock.TestingT
	Cleanup(func())
}

// newMockMinioClient creates a new instance of mockMinioClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func newMockMinioClient(t mockConstructorTestingTnewMockMinioClient) *mockMinioClient {
	mock := &mockMinioClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
