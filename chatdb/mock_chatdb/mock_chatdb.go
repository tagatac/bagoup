// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/tagatac/bagoup/chatdb (interfaces: ChatDB)

// Package mock_chatdb is a generated GoMock package.
package mock_chatdb

import (
	reflect "reflect"

	semver "github.com/Masterminds/semver"
	vcard "github.com/emersion/go-vcard"
	gomock "github.com/golang/mock/gomock"
	chatdb "github.com/tagatac/bagoup/chatdb"
	pathtools "github.com/tagatac/bagoup/pathtools"
)

// MockChatDB is a mock of ChatDB interface.
type MockChatDB struct {
	ctrl     *gomock.Controller
	recorder *MockChatDBMockRecorder
}

// MockChatDBMockRecorder is the mock recorder for MockChatDB.
type MockChatDBMockRecorder struct {
	mock *MockChatDB
}

// NewMockChatDB creates a new mock instance.
func NewMockChatDB(ctrl *gomock.Controller) *MockChatDB {
	mock := &MockChatDB{ctrl: ctrl}
	mock.recorder = &MockChatDBMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockChatDB) EXPECT() *MockChatDBMockRecorder {
	return m.recorder
}

// GetAttachmentPaths mocks base method.
func (m *MockChatDB) GetAttachmentPaths(arg0 pathtools.PathTools) (map[int][]chatdb.Attachment, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAttachmentPaths", arg0)
	ret0, _ := ret[0].(map[int][]chatdb.Attachment)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAttachmentPaths indicates an expected call of GetAttachmentPaths.
func (mr *MockChatDBMockRecorder) GetAttachmentPaths(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAttachmentPaths", reflect.TypeOf((*MockChatDB)(nil).GetAttachmentPaths), arg0)
}

// GetChats mocks base method.
func (m *MockChatDB) GetChats(arg0 map[string]*vcard.Card) ([]chatdb.EntityChats, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetChats", arg0)
	ret0, _ := ret[0].([]chatdb.EntityChats)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetChats indicates an expected call of GetChats.
func (mr *MockChatDBMockRecorder) GetChats(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetChats", reflect.TypeOf((*MockChatDB)(nil).GetChats), arg0)
}

// GetHandleMap mocks base method.
func (m *MockChatDB) GetHandleMap(arg0 map[string]*vcard.Card) (map[int]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetHandleMap", arg0)
	ret0, _ := ret[0].(map[int]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetHandleMap indicates an expected call of GetHandleMap.
func (mr *MockChatDBMockRecorder) GetHandleMap(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHandleMap", reflect.TypeOf((*MockChatDB)(nil).GetHandleMap), arg0)
}

// GetMessage mocks base method.
func (m *MockChatDB) GetMessage(arg0 int, arg1 map[int]string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMessage", arg0, arg1)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetMessage indicates an expected call of GetMessage.
func (mr *MockChatDBMockRecorder) GetMessage(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMessage", reflect.TypeOf((*MockChatDB)(nil).GetMessage), arg0, arg1)
}

// GetMessageIDs mocks base method.
func (m *MockChatDB) GetMessageIDs(arg0 int) ([]chatdb.DatedMessageID, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMessageIDs", arg0)
	ret0, _ := ret[0].([]chatdb.DatedMessageID)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetMessageIDs indicates an expected call of GetMessageIDs.
func (mr *MockChatDBMockRecorder) GetMessageIDs(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMessageIDs", reflect.TypeOf((*MockChatDB)(nil).GetMessageIDs), arg0)
}

// Init mocks base method.
func (m *MockChatDB) Init(arg0 *semver.Version) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Init", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Init indicates an expected call of Init.
func (mr *MockChatDBMockRecorder) Init(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Init", reflect.TypeOf((*MockChatDB)(nil).Init), arg0)
}
