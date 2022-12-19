// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/conduitio-labs/notionapi (interfaces: BlockService)

// Package mock is a generated GoMock package.
package mock

import (
	context "context"
	reflect "reflect"

	notionapi "github.com/conduitio-labs/notionapi"
	gomock "github.com/golang/mock/gomock"
)

// BlockService is a mock of BlockService interface.
type BlockService struct {
	ctrl     *gomock.Controller
	recorder *BlockServiceMockRecorder
}

// BlockServiceMockRecorder is the mock recorder for BlockService.
type BlockServiceMockRecorder struct {
	mock *BlockService
}

// NewBlockService creates a new mock instance.
func NewBlockService(ctrl *gomock.Controller) *BlockService {
	mock := &BlockService{ctrl: ctrl}
	mock.recorder = &BlockServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *BlockService) EXPECT() *BlockServiceMockRecorder {
	return m.recorder
}

// AppendChildren mocks base method.
func (m *BlockService) AppendChildren(arg0 context.Context, arg1 notionapi.BlockID, arg2 *notionapi.AppendBlockChildrenRequest) (*notionapi.AppendBlockChildrenResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AppendChildren", arg0, arg1, arg2)
	ret0, _ := ret[0].(*notionapi.AppendBlockChildrenResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AppendChildren indicates an expected call of AppendChildren.
func (mr *BlockServiceMockRecorder) AppendChildren(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AppendChildren", reflect.TypeOf((*BlockService)(nil).AppendChildren), arg0, arg1, arg2)
}

// Delete mocks base method.
func (m *BlockService) Delete(arg0 context.Context, arg1 notionapi.BlockID) (notionapi.Block, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", arg0, arg1)
	ret0, _ := ret[0].(notionapi.Block)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Delete indicates an expected call of Delete.
func (mr *BlockServiceMockRecorder) Delete(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*BlockService)(nil).Delete), arg0, arg1)
}

// Get mocks base method.
func (m *BlockService) Get(arg0 context.Context, arg1 notionapi.BlockID) (notionapi.Block, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0, arg1)
	ret0, _ := ret[0].(notionapi.Block)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *BlockServiceMockRecorder) Get(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*BlockService)(nil).Get), arg0, arg1)
}

// GetChildren mocks base method.
func (m *BlockService) GetChildren(arg0 context.Context, arg1 notionapi.BlockID, arg2 *notionapi.Pagination) (*notionapi.GetChildrenResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetChildren", arg0, arg1, arg2)
	ret0, _ := ret[0].(*notionapi.GetChildrenResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetChildren indicates an expected call of GetChildren.
func (mr *BlockServiceMockRecorder) GetChildren(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetChildren", reflect.TypeOf((*BlockService)(nil).GetChildren), arg0, arg1, arg2)
}

// Update mocks base method.
func (m *BlockService) Update(arg0 context.Context, arg1 notionapi.BlockID, arg2 *notionapi.BlockUpdateRequest) (notionapi.Block, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", arg0, arg1, arg2)
	ret0, _ := ret[0].(notionapi.Block)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Update indicates an expected call of Update.
func (mr *BlockServiceMockRecorder) Update(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*BlockService)(nil).Update), arg0, arg1, arg2)
}