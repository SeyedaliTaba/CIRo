// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/scionproto/scion/pkg/private/xtest (interfaces: Callback)

// Package mock_xtest is a generated GoMock package.
package mock_xtest

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockCallback is a mock of Callback interface.
type MockCallback struct {
	ctrl     *gomock.Controller
	recorder *MockCallbackMockRecorder
}

// MockCallbackMockRecorder is the mock recorder for MockCallback.
type MockCallbackMockRecorder struct {
	mock *MockCallback
}

// NewMockCallback creates a new mock instance.
func NewMockCallback(ctrl *gomock.Controller) *MockCallback {
	mock := &MockCallback{ctrl: ctrl}
	mock.recorder = &MockCallbackMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCallback) EXPECT() *MockCallbackMockRecorder {
	return m.recorder
}

// Call mocks base method.
func (m *MockCallback) Call() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Call")
}

// Call indicates an expected call of Call.
func (mr *MockCallbackMockRecorder) Call() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Call", reflect.TypeOf((*MockCallback)(nil).Call))
}
