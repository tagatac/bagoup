// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/tagatac/bagoup/opsys (interfaces: OS)

// Package mock_opsys is a generated GoMock package.
package mock_opsys

import (
	semver "github.com/Masterminds/semver"
	vcard "github.com/emersion/go-vcard"
	gomock "github.com/golang/mock/gomock"
	chatdb "github.com/tagatac/bagoup/chatdb"
	reflect "reflect"
)

// MockOS is a mock of OS interface
type MockOS struct {
	ctrl     *gomock.Controller
	recorder *MockOSMockRecorder
}

// MockOSMockRecorder is the mock recorder for MockOS
type MockOSMockRecorder struct {
	mock *MockOS
}

// NewMockOS creates a new mock instance
func NewMockOS(ctrl *gomock.Controller) *MockOS {
	mock := &MockOS{ctrl: ctrl}
	mock.recorder = &MockOSMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockOS) EXPECT() *MockOSMockRecorder {
	return m.recorder
}

// ExportChats mocks base method
func (m *MockOS) ExportChats(arg0 chatdb.ChatDB, arg1 string, arg2 map[string]*vcard.Card, arg3 map[int]string, arg4 *semver.Version) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ExportChats", arg0, arg1, arg2, arg3, arg4)
	ret0, _ := ret[0].(error)
	return ret0
}

// ExportChats indicates an expected call of ExportChats
func (mr *MockOSMockRecorder) ExportChats(arg0, arg1, arg2, arg3, arg4 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ExportChats", reflect.TypeOf((*MockOS)(nil).ExportChats), arg0, arg1, arg2, arg3, arg4)
}

// FileExist mocks base method
func (m *MockOS) FileExist(arg0 string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FileExist", arg0)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FileExist indicates an expected call of FileExist
func (mr *MockOSMockRecorder) FileExist(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FileExist", reflect.TypeOf((*MockOS)(nil).FileExist), arg0)
}

// GetContactMap mocks base method
func (m *MockOS) GetContactMap(arg0 string) (map[string]*vcard.Card, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetContactMap", arg0)
	ret0, _ := ret[0].(map[string]*vcard.Card)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetContactMap indicates an expected call of GetContactMap
func (mr *MockOSMockRecorder) GetContactMap(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetContactMap", reflect.TypeOf((*MockOS)(nil).GetContactMap), arg0)
}

// GetMacOSVersion mocks base method
func (m *MockOS) GetMacOSVersion() (*semver.Version, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMacOSVersion")
	ret0, _ := ret[0].(*semver.Version)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetMacOSVersion indicates an expected call of GetMacOSVersion
func (mr *MockOSMockRecorder) GetMacOSVersion() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMacOSVersion", reflect.TypeOf((*MockOS)(nil).GetMacOSVersion))
}
