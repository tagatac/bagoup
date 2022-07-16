// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/tagatac/bagoup/opsys (interfaces: OS)

// Package mock_opsys is a generated GoMock package.
package mock_opsys

import (
	fs "io/fs"
	reflect "reflect"
	time "time"

	semver "github.com/Masterminds/semver"
	vcard "github.com/emersion/go-vcard"
	gomock "github.com/golang/mock/gomock"
	afero "github.com/spf13/afero"
	opsys "github.com/tagatac/bagoup/opsys"
)

// MockOS is a mock of OS interface.
type MockOS struct {
	ctrl     *gomock.Controller
	recorder *MockOSMockRecorder
}

// MockOSMockRecorder is the mock recorder for MockOS.
type MockOSMockRecorder struct {
	mock *MockOS
}

// NewMockOS creates a new mock instance.
func NewMockOS(ctrl *gomock.Controller) *MockOS {
	mock := &MockOS{ctrl: ctrl}
	mock.recorder = &MockOSMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockOS) EXPECT() *MockOSMockRecorder {
	return m.recorder
}

// Chmod mocks base method.
func (m *MockOS) Chmod(arg0 string, arg1 fs.FileMode) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Chmod", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Chmod indicates an expected call of Chmod.
func (mr *MockOSMockRecorder) Chmod(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Chmod", reflect.TypeOf((*MockOS)(nil).Chmod), arg0, arg1)
}

// Chown mocks base method.
func (m *MockOS) Chown(arg0 string, arg1, arg2 int) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Chown", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// Chown indicates an expected call of Chown.
func (mr *MockOSMockRecorder) Chown(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Chown", reflect.TypeOf((*MockOS)(nil).Chown), arg0, arg1, arg2)
}

// Chtimes mocks base method.
func (m *MockOS) Chtimes(arg0 string, arg1, arg2 time.Time) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Chtimes", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// Chtimes indicates an expected call of Chtimes.
func (mr *MockOSMockRecorder) Chtimes(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Chtimes", reflect.TypeOf((*MockOS)(nil).Chtimes), arg0, arg1, arg2)
}

// CopyFile mocks base method.
func (m *MockOS) CopyFile(arg0, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CopyFile", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// CopyFile indicates an expected call of CopyFile.
func (mr *MockOSMockRecorder) CopyFile(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CopyFile", reflect.TypeOf((*MockOS)(nil).CopyFile), arg0, arg1)
}

// Create mocks base method.
func (m *MockOS) Create(arg0 string) (afero.File, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", arg0)
	ret0, _ := ret[0].(afero.File)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *MockOSMockRecorder) Create(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockOS)(nil).Create), arg0)
}

// FileAccess mocks base method.
func (m *MockOS) FileAccess(arg0 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FileAccess", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// FileAccess indicates an expected call of FileAccess.
func (mr *MockOSMockRecorder) FileAccess(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FileAccess", reflect.TypeOf((*MockOS)(nil).FileAccess), arg0)
}

// FileExist mocks base method.
func (m *MockOS) FileExist(arg0 string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FileExist", arg0)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FileExist indicates an expected call of FileExist.
func (mr *MockOSMockRecorder) FileExist(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FileExist", reflect.TypeOf((*MockOS)(nil).FileExist), arg0)
}

// GetContactMap mocks base method.
func (m *MockOS) GetContactMap(arg0 string) (map[string]*vcard.Card, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetContactMap", arg0)
	ret0, _ := ret[0].(map[string]*vcard.Card)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetContactMap indicates an expected call of GetContactMap.
func (mr *MockOSMockRecorder) GetContactMap(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetContactMap", reflect.TypeOf((*MockOS)(nil).GetContactMap), arg0)
}

// GetMacOSVersion mocks base method.
func (m *MockOS) GetMacOSVersion() (*semver.Version, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMacOSVersion")
	ret0, _ := ret[0].(*semver.Version)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetMacOSVersion indicates an expected call of GetMacOSVersion.
func (mr *MockOSMockRecorder) GetMacOSVersion() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMacOSVersion", reflect.TypeOf((*MockOS)(nil).GetMacOSVersion))
}

// GetOpenFilesLimit mocks base method.
func (m *MockOS) GetOpenFilesLimit() int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOpenFilesLimit")
	ret0, _ := ret[0].(int)
	return ret0
}

// GetOpenFilesLimit indicates an expected call of GetOpenFilesLimit.
func (mr *MockOSMockRecorder) GetOpenFilesLimit() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOpenFilesLimit", reflect.TypeOf((*MockOS)(nil).GetOpenFilesLimit))
}

// HEIC2JPG mocks base method.
func (m *MockOS) HEIC2JPG(arg0 string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HEIC2JPG", arg0)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// HEIC2JPG indicates an expected call of HEIC2JPG.
func (mr *MockOSMockRecorder) HEIC2JPG(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HEIC2JPG", reflect.TypeOf((*MockOS)(nil).HEIC2JPG), arg0)
}

// Mkdir mocks base method.
func (m *MockOS) Mkdir(arg0 string, arg1 fs.FileMode) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Mkdir", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Mkdir indicates an expected call of Mkdir.
func (mr *MockOSMockRecorder) Mkdir(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Mkdir", reflect.TypeOf((*MockOS)(nil).Mkdir), arg0, arg1)
}

// MkdirAll mocks base method.
func (m *MockOS) MkdirAll(arg0 string, arg1 fs.FileMode) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MkdirAll", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// MkdirAll indicates an expected call of MkdirAll.
func (mr *MockOSMockRecorder) MkdirAll(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MkdirAll", reflect.TypeOf((*MockOS)(nil).MkdirAll), arg0, arg1)
}

// Name mocks base method.
func (m *MockOS) Name() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Name")
	ret0, _ := ret[0].(string)
	return ret0
}

// Name indicates an expected call of Name.
func (mr *MockOSMockRecorder) Name() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Name", reflect.TypeOf((*MockOS)(nil).Name))
}

// NewOutFile mocks base method.
func (m *MockOS) NewOutFile(arg0 string, arg1, arg2 bool) (opsys.OutFile, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewOutFile", arg0, arg1, arg2)
	ret0, _ := ret[0].(opsys.OutFile)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewOutFile indicates an expected call of NewOutFile.
func (mr *MockOSMockRecorder) NewOutFile(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewOutFile", reflect.TypeOf((*MockOS)(nil).NewOutFile), arg0, arg1, arg2)
}

// Open mocks base method.
func (m *MockOS) Open(arg0 string) (afero.File, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Open", arg0)
	ret0, _ := ret[0].(afero.File)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Open indicates an expected call of Open.
func (mr *MockOSMockRecorder) Open(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Open", reflect.TypeOf((*MockOS)(nil).Open), arg0)
}

// OpenFile mocks base method.
func (m *MockOS) OpenFile(arg0 string, arg1 int, arg2 fs.FileMode) (afero.File, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "OpenFile", arg0, arg1, arg2)
	ret0, _ := ret[0].(afero.File)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// OpenFile indicates an expected call of OpenFile.
func (mr *MockOSMockRecorder) OpenFile(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OpenFile", reflect.TypeOf((*MockOS)(nil).OpenFile), arg0, arg1, arg2)
}

// Remove mocks base method.
func (m *MockOS) Remove(arg0 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Remove", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Remove indicates an expected call of Remove.
func (mr *MockOSMockRecorder) Remove(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Remove", reflect.TypeOf((*MockOS)(nil).Remove), arg0)
}

// RemoveAll mocks base method.
func (m *MockOS) RemoveAll(arg0 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoveAll", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// RemoveAll indicates an expected call of RemoveAll.
func (mr *MockOSMockRecorder) RemoveAll(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveAll", reflect.TypeOf((*MockOS)(nil).RemoveAll), arg0)
}

// Rename mocks base method.
func (m *MockOS) Rename(arg0, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Rename", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Rename indicates an expected call of Rename.
func (mr *MockOSMockRecorder) Rename(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Rename", reflect.TypeOf((*MockOS)(nil).Rename), arg0, arg1)
}

// ResetOpenFilesLimit mocks base method.
func (m *MockOS) ResetOpenFilesLimit() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ResetOpenFilesLimit")
	ret0, _ := ret[0].(error)
	return ret0
}

// ResetOpenFilesLimit indicates an expected call of ResetOpenFilesLimit.
func (mr *MockOSMockRecorder) ResetOpenFilesLimit() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ResetOpenFilesLimit", reflect.TypeOf((*MockOS)(nil).ResetOpenFilesLimit))
}

// RmTempDir mocks base method.
func (m *MockOS) RmTempDir() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RmTempDir")
	ret0, _ := ret[0].(error)
	return ret0
}

// RmTempDir indicates an expected call of RmTempDir.
func (mr *MockOSMockRecorder) RmTempDir() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RmTempDir", reflect.TypeOf((*MockOS)(nil).RmTempDir))
}

// SetOpenFilesLimit mocks base method.
func (m *MockOS) SetOpenFilesLimit(arg0 int) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetOpenFilesLimit", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetOpenFilesLimit indicates an expected call of SetOpenFilesLimit.
func (mr *MockOSMockRecorder) SetOpenFilesLimit(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetOpenFilesLimit", reflect.TypeOf((*MockOS)(nil).SetOpenFilesLimit), arg0)
}

// Stat mocks base method.
func (m *MockOS) Stat(arg0 string) (fs.FileInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Stat", arg0)
	ret0, _ := ret[0].(fs.FileInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Stat indicates an expected call of Stat.
func (mr *MockOSMockRecorder) Stat(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stat", reflect.TypeOf((*MockOS)(nil).Stat), arg0)
}
