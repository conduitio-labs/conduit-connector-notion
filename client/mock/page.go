// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/conduitio-labs/notionapi (interfaces: PageService)

// Package mock is a generated GoMock package.
package mock

import (
	context "context"
	reflect "reflect"

	notionapi "github.com/conduitio-labs/notionapi"
	gomock "github.com/golang/mock/gomock"
)

// PageService is a mock of PageService interface.
type PageService struct {
	ctrl     *gomock.Controller
	recorder *PageServiceMockRecorder
}

// PageServiceMockRecorder is the mock recorder for PageService.
type PageServiceMockRecorder struct {
	mock *PageService
}

// NewPageService creates a new mock instance.
func NewPageService(ctrl *gomock.Controller) *PageService {
	mock := &PageService{ctrl: ctrl}
	mock.recorder = &PageServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *PageService) EXPECT() *PageServiceMockRecorder {
	return m.recorder
}

// Create mocks base method.
func (m *PageService) Create(arg0 context.Context, arg1 *notionapi.PageCreateRequest) (*notionapi.Page, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", arg0, arg1)
	ret0, _ := ret[0].(*notionapi.Page)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *PageServiceMockRecorder) Create(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*PageService)(nil).Create), arg0, arg1)
}

// Get mocks base method.
func (m *PageService) Get(arg0 context.Context, arg1 notionapi.PageID) (*notionapi.Page, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0, arg1)
	ret0, _ := ret[0].(*notionapi.Page)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *PageServiceMockRecorder) Get(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*PageService)(nil).Get), arg0, arg1)
}

// Update mocks base method.
func (m *PageService) Update(arg0 context.Context, arg1 notionapi.PageID, arg2 *notionapi.PageUpdateRequest) (*notionapi.Page, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", arg0, arg1, arg2)
	ret0, _ := ret[0].(*notionapi.Page)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Update indicates an expected call of Update.
func (mr *PageServiceMockRecorder) Update(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*PageService)(nil).Update), arg0, arg1, arg2)
}