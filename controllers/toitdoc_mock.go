// Code generated by MockGen. DO NOT EDIT.
// Source: controllers/toitdoc.go

// Package controllers is a generated GoMock package.
package controllers

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	toitdoc "github.com/toitware/tpkg.git/pkg/toitdoc"
	tpkg "github.com/toitware/tpkg.git/pkg/tpkg"
)

// MockToitdoc is a mock of Toitdoc interface.
type MockToitdoc struct {
	ctrl     *gomock.Controller
	recorder *MockToitdocMockRecorder
}

// MockToitdocMockRecorder is the mock recorder for MockToitdoc.
type MockToitdocMockRecorder struct {
	mock *MockToitdoc
}

// NewMockToitdoc creates a new mock instance.
func NewMockToitdoc(ctrl *gomock.Controller) *MockToitdoc {
	mock := &MockToitdoc{ctrl: ctrl}
	mock.recorder = &MockToitdocMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockToitdoc) EXPECT() *MockToitdocMockRecorder {
	return m.recorder
}

// Load mocks base method.
func (m *MockToitdoc) Load(ctx context.Context, desc *tpkg.Desc) (toitdoc.Doc, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Load", ctx, desc)
	ret0, _ := ret[0].(toitdoc.Doc)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Load indicates an expected call of Load.
func (mr *MockToitdocMockRecorder) Load(ctx, desc interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Load", reflect.TypeOf((*MockToitdoc)(nil).Load), ctx, desc)
}
