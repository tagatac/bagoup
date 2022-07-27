// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/tagatac/bagoup/opsys (interfaces: OutFile)

// Package mock_opsys is a generated GoMock package.
package mock_opsys

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockOutFile is a mock of OutFile interface.
type MockOutFile struct {
	ctrl     *gomock.Controller
	recorder *MockOutFileMockRecorder
}

// MockOutFileMockRecorder is the mock recorder for MockOutFile.
type MockOutFileMockRecorder struct {
	mock *MockOutFile
}

// NewMockOutFile creates a new mock instance.
func NewMockOutFile(ctrl *gomock.Controller) *MockOutFile {
	mock := &MockOutFile{ctrl: ctrl}
	mock.recorder = &MockOutFileMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockOutFile) EXPECT() *MockOutFileMockRecorder {
	return m.recorder
}

// Flush mocks base method.
func (m *MockOutFile) Flush() (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Flush")
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Flush indicates an expected call of Flush.
func (mr *MockOutFileMockRecorder) Flush() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Flush", reflect.TypeOf((*MockOutFile)(nil).Flush))
}

// Name mocks base method.
func (m *MockOutFile) Name() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Name")
	ret0, _ := ret[0].(string)
	return ret0
}

// Name indicates an expected call of Name.
func (mr *MockOutFileMockRecorder) Name() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Name", reflect.TypeOf((*MockOutFile)(nil).Name))
}

// ReferenceAttachment mocks base method.
func (m *MockOutFile) ReferenceAttachment(arg0 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReferenceAttachment", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// ReferenceAttachment indicates an expected call of ReferenceAttachment.
func (mr *MockOutFileMockRecorder) ReferenceAttachment(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReferenceAttachment", reflect.TypeOf((*MockOutFile)(nil).ReferenceAttachment), arg0)
}

// WriteAttachment mocks base method.
func (m *MockOutFile) WriteAttachment(arg0 string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WriteAttachment", arg0)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WriteAttachment indicates an expected call of WriteAttachment.
func (mr *MockOutFileMockRecorder) WriteAttachment(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WriteAttachment", reflect.TypeOf((*MockOutFile)(nil).WriteAttachment), arg0)
}

// WriteMessage mocks base method.
func (m *MockOutFile) WriteMessage(arg0 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WriteMessage", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// WriteMessage indicates an expected call of WriteMessage.
func (mr *MockOutFileMockRecorder) WriteMessage(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WriteMessage", reflect.TypeOf((*MockOutFile)(nil).WriteMessage), arg0)
}
