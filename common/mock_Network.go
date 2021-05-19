// Code generated by mockery v1.1.0. DO NOT EDIT.

package common

import (
	context "context"
	io "io"

	mock "github.com/stretchr/testify/mock"
)

// MockNetwork is an autogenerated mock type for the Network type
type MockNetwork struct {
	mock.Mock
}

// DownloadArtifacts provides a mock function with given fields: config, artifactsFile, directDownload
func (_m *MockNetwork) DownloadArtifacts(config JobCredentials, artifactsFile io.WriteCloser, directDownload *bool) DownloadState {
	ret := _m.Called(config, artifactsFile, directDownload)

	var r0 DownloadState
	if rf, ok := ret.Get(0).(func(JobCredentials, io.WriteCloser, *bool) DownloadState); ok {
		r0 = rf(config, artifactsFile, directDownload)
	} else {
		r0 = ret.Get(0).(DownloadState)
	}

	return r0
}

// PatchTrace provides a mock function with given fields: config, jobCredentials, r, offset, length
func (_m *MockNetwork) PatchTrace(config RunnerConfig, jobCredentials *JobCredentials, r io.Reader, offset int, length int) PatchTraceResult {
	ret := _m.Called(config, jobCredentials, r, offset, length)

	var r0 PatchTraceResult
	if rf, ok := ret.Get(0).(func(RunnerConfig, *JobCredentials, io.Reader, int, int) PatchTraceResult); ok {
		r0 = rf(config, jobCredentials, r, offset, length)
	} else {
		r0 = ret.Get(0).(PatchTraceResult)
	}

	return r0
}

// ProcessJob provides a mock function with given fields: config, buildCredentials
func (_m *MockNetwork) ProcessJob(config RunnerConfig, buildCredentials *JobCredentials) (JobTrace, error) {
	ret := _m.Called(config, buildCredentials)

	var r0 JobTrace
	if rf, ok := ret.Get(0).(func(RunnerConfig, *JobCredentials) JobTrace); ok {
		r0 = rf(config, buildCredentials)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(JobTrace)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(RunnerConfig, *JobCredentials) error); ok {
		r1 = rf(config, buildCredentials)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RegisterRunner provides a mock function with given fields: config, parameters
func (_m *MockNetwork) RegisterRunner(config RunnerCredentials, parameters RegisterRunnerParameters) *RegisterRunnerResponse {
	ret := _m.Called(config, parameters)

	var r0 *RegisterRunnerResponse
	if rf, ok := ret.Get(0).(func(RunnerCredentials, RegisterRunnerParameters) *RegisterRunnerResponse); ok {
		r0 = rf(config, parameters)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*RegisterRunnerResponse)
		}
	}

	return r0
}

// RequestJob provides a mock function with given fields: ctx, config, sessionInfo
func (_m *MockNetwork) RequestJob(ctx context.Context, config RunnerConfig, sessionInfo *SessionInfo) (*JobResponse, bool) {
	ret := _m.Called(ctx, config, sessionInfo)

	var r0 *JobResponse
	if rf, ok := ret.Get(0).(func(context.Context, RunnerConfig, *SessionInfo) *JobResponse); ok {
		r0 = rf(ctx, config, sessionInfo)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*JobResponse)
		}
	}

	var r1 bool
	if rf, ok := ret.Get(1).(func(context.Context, RunnerConfig, *SessionInfo) bool); ok {
		r1 = rf(ctx, config, sessionInfo)
	} else {
		r1 = ret.Get(1).(bool)
	}

	return r0, r1
}

// UnregisterRunner provides a mock function with given fields: config
func (_m *MockNetwork) UnregisterRunner(config RunnerCredentials) bool {
	ret := _m.Called(config)

	var r0 bool
	if rf, ok := ret.Get(0).(func(RunnerCredentials) bool); ok {
		r0 = rf(config)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// UpdateJob provides a mock function with given fields: config, jobCredentials, jobInfo
func (_m *MockNetwork) UpdateJob(config RunnerConfig, jobCredentials *JobCredentials, jobInfo UpdateJobInfo) UpdateJobResult {
	ret := _m.Called(config, jobCredentials, jobInfo)

	var r0 UpdateJobResult
	if rf, ok := ret.Get(0).(func(RunnerConfig, *JobCredentials, UpdateJobInfo) UpdateJobResult); ok {
		r0 = rf(config, jobCredentials, jobInfo)
	} else {
		r0 = ret.Get(0).(UpdateJobResult)
	}

	return r0
}

// UploadRawArtifacts provides a mock function with given fields: config, reader, options
func (_m *MockNetwork) UploadRawArtifacts(config JobCredentials, reader io.ReadCloser, options ArtifactsOptions) (UploadState, string) {
	ret := _m.Called(config, reader, options)

	var r0 UploadState
	if rf, ok := ret.Get(0).(func(JobCredentials, io.ReadCloser, ArtifactsOptions) UploadState); ok {
		r0 = rf(config, reader, options)
	} else {
		r0 = ret.Get(0).(UploadState)
	}

	var r1 string
	if rf, ok := ret.Get(1).(func(JobCredentials, io.ReadCloser, ArtifactsOptions) string); ok {
		r1 = rf(config, reader, options)
	} else {
		r1 = ret.Get(1).(string)
	}

	return r0, r1
}

// VerifyRunner provides a mock function with given fields: config
func (_m *MockNetwork) VerifyRunner(config RunnerCredentials) bool {
	ret := _m.Called(config)

	var r0 bool
	if rf, ok := ret.Get(0).(func(RunnerCredentials) bool); ok {
		r0 = rf(config)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}
