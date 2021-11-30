// Code generated by MockGen. DO NOT EDIT.
// Source: ./interface.go

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	types "github.com/spacemeshos/go-spacemesh/common/types"
)

// MockblockOracle is a mock of blockOracle interface.
type MockblockOracle struct {
	ctrl     *gomock.Controller
	recorder *MockblockOracleMockRecorder
}

// MockblockOracleMockRecorder is the mock recorder for MockblockOracle.
type MockblockOracleMockRecorder struct {
	mock *MockblockOracle
}

// NewMockblockOracle creates a new mock instance.
func NewMockblockOracle(ctrl *gomock.Controller) *MockblockOracle {
	mock := &MockblockOracle{ctrl: ctrl}
	mock.recorder = &MockblockOracleMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockblockOracle) EXPECT() *MockblockOracleMockRecorder {
	return m.recorder
}

// BlockEligible mocks base method.
func (m *MockblockOracle) BlockEligible(arg0 types.LayerID) (types.ATXID, []types.BlockEligibilityProof, []types.ATXID, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BlockEligible", arg0)
	ret0, _ := ret[0].(types.ATXID)
	ret1, _ := ret[1].([]types.BlockEligibilityProof)
	ret2, _ := ret[2].([]types.ATXID)
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// BlockEligible indicates an expected call of BlockEligible.
func (mr *MockblockOracleMockRecorder) BlockEligible(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BlockEligible", reflect.TypeOf((*MockblockOracle)(nil).BlockEligible), arg0)
}

// Mockprojector is a mock of projector interface.
type Mockprojector struct {
	ctrl     *gomock.Controller
	recorder *MockprojectorMockRecorder
}

// MockprojectorMockRecorder is the mock recorder for Mockprojector.
type MockprojectorMockRecorder struct {
	mock *Mockprojector
}

// NewMockprojector creates a new mock instance.
func NewMockprojector(ctrl *gomock.Controller) *Mockprojector {
	mock := &Mockprojector{ctrl: ctrl}
	mock.recorder = &MockprojectorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *Mockprojector) EXPECT() *MockprojectorMockRecorder {
	return m.recorder
}

// GetProjection mocks base method.
func (m *Mockprojector) GetProjection(arg0 types.Address) (uint64, uint64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetProjection", arg0)
	ret0, _ := ret[0].(uint64)
	ret1, _ := ret[1].(uint64)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// GetProjection indicates an expected call of GetProjection.
func (mr *MockprojectorMockRecorder) GetProjection(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetProjection", reflect.TypeOf((*Mockprojector)(nil).GetProjection), arg0)
}