// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/tagatac/bagoup/opsys (interfaces: OS)

// Package mock_opsys is a generated GoMock package.
package mock_opsys

import (
	semver "github.com/Masterminds/semver"
	vcard "github.com/emersion/go-vcard"
	gomock "github.com/golang/mock/gomock"
	afero "github.com/spf13/afero"
	os "os"
	reflect "reflect"
	time "time"
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

// Chmod mocks base method
func (m *MockOS) Chmod(arg0 string, arg1 os.FileMode) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Chmod", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Chmod indicates an expected call of Chmod
func (mr *MockOSMockRecorder) Chmod(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Chmod", reflect.TypeOf((*MockOS)(nil).Chmod), arg0, arg1)
}

// Chtimes mocks base method
func (m *MockOS) Chtimes(arg0 string, arg1, arg2 time.Time) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Chtimes", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// Chtimes indicates an expected call of Chtimes
func (mr *MockOSMockRecorder) Chtimes(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Chtimes", reflect.TypeOf((*MockOS)(nil).Chtimes), arg0, arg1, arg2)
}

// Create mocks base method
func (m *MockOS) Create(arg0 string) (afero.File, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", arg0)
	ret0, _ := ret[0].(afero.File)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create
func (mr *MockOSMockRecorder) Create(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockOS)(nil).Create), arg0)
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

// Mkdir mocks base method
func (m *MockOS) Mkdir(arg0 string, arg1 os.FileMode) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Mkdir", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Mkdir indicates an expected call of Mkdir
func (mr *MockOSMockRecorder) Mkdir(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Mkdir", reflect.TypeOf((*MockOS)(nil).Mkdir), arg0, arg1)
}

// MkdirAll mocks base method
func (m *MockOS) MkdirAll(arg0 string, arg1 os.FileMode) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MkdirAll", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// MkdirAll indicates an expected call of MkdirAll
func (mr *MockOSMockRecorder) MkdirAll(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MkdirAll", reflect.TypeOf((*MockOS)(nil).MkdirAll), arg0, arg1)
}

// Name mocks base method
func (m *MockOS) Name() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Name")
	ret0, _ := ret[0].(string)
	return ret0
}

// Name indicates an expected call of Name
func (mr *MockOSMockRecorder) Name() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Name", reflect.TypeOf((*MockOS)(nil).Name))
}

// Open mocks base method
func (m *MockOS) Open(arg0 string) (afero.File, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Open", arg0)
	ret0, _ := ret[0].(afero.File)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Open indicates an expected call of Open
func (mr *MockOSMockRecorder) Open(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Open", reflect.TypeOf((*MockOS)(nil).Open), arg0)
}

// OpenFile mocks base method
func (m *MockOS) OpenFile(arg0 string, arg1 int, arg2 os.FileMode) (afero.File, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "OpenFile", arg0, arg1, arg2)
	ret0, _ := ret[0].(afero.File)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// OpenFile indicates an expected call of OpenFile
func (mr *MockOSMockRecorder) OpenFile(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OpenFile", reflect.TypeOf((*MockOS)(nil).OpenFile), arg0, arg1, arg2)
}

// Remove mocks base method
func (m *MockOS) Remove(arg0 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Remove", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Remove indicates an expected call of Remove
func (mr *MockOSMockRecorder) Remove(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Remove", reflect.TypeOf((*MockOS)(nil).Remove), arg0)
}

// RemoveAll mocks base method
func (m *MockOS) RemoveAll(arg0 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoveAll", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// RemoveAll indicates an expected call of RemoveAll
func (mr *MockOSMockRecorder) RemoveAll(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveAll", reflect.TypeOf((*MockOS)(nil).RemoveAll), arg0)
}

// Rename mocks base method
func (m *MockOS) Rename(arg0, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Rename", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Rename indicates an expected call of Rename
func (mr *MockOSMockRecorder) Rename(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Rename", reflect.TypeOf((*MockOS)(nil).Rename), arg0, arg1)
}

// Stat mocks base method
func (m *MockOS) Stat(arg0 string) (os.FileInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Stat", arg0)
	ret0, _ := ret[0].(os.FileInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Stat indicates an expected call of Stat
func (mr *MockOSMockRecorder) Stat(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stat", reflect.TypeOf((*MockOS)(nil).Stat), arg0)
}