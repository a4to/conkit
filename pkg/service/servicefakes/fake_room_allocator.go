// Code generated by counterfeiter. DO NOT EDIT.
package servicefakes

import (
	"context"
	"sync"

	"github.com/a4to/conkit-server/pkg/service"
	"github.com/a4to/protocol/conkit"
)

type FakeRoomAllocator struct {
	AutoCreateEnabledStub        func(context.Context) bool
	autoCreateEnabledMutex       sync.RWMutex
	autoCreateEnabledArgsForCall []struct {
		arg1 context.Context
	}
	autoCreateEnabledReturns struct {
		result1 bool
	}
	autoCreateEnabledReturnsOnCall map[int]struct {
		result1 bool
	}
	CreateRoomStub        func(context.Context, *conkit.CreateRoomRequest, bool) (*conkit.Room, *conkit.RoomInternal, bool, error)
	createRoomMutex       sync.RWMutex
	createRoomArgsForCall []struct {
		arg1 context.Context
		arg2 *conkit.CreateRoomRequest
		arg3 bool
	}
	createRoomReturns struct {
		result1 *conkit.Room
		result2 *conkit.RoomInternal
		result3 bool
		result4 error
	}
	createRoomReturnsOnCall map[int]struct {
		result1 *conkit.Room
		result2 *conkit.RoomInternal
		result3 bool
		result4 error
	}
	SelectRoomNodeStub        func(context.Context, conkit.RoomName, conkit.NodeID) error
	selectRoomNodeMutex       sync.RWMutex
	selectRoomNodeArgsForCall []struct {
		arg1 context.Context
		arg2 conkit.RoomName
		arg3 conkit.NodeID
	}
	selectRoomNodeReturns struct {
		result1 error
	}
	selectRoomNodeReturnsOnCall map[int]struct {
		result1 error
	}
	ValidateCreateRoomStub        func(context.Context, conkit.RoomName) error
	validateCreateRoomMutex       sync.RWMutex
	validateCreateRoomArgsForCall []struct {
		arg1 context.Context
		arg2 conkit.RoomName
	}
	validateCreateRoomReturns struct {
		result1 error
	}
	validateCreateRoomReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeRoomAllocator) AutoCreateEnabled(arg1 context.Context) bool {
	fake.autoCreateEnabledMutex.Lock()
	ret, specificReturn := fake.autoCreateEnabledReturnsOnCall[len(fake.autoCreateEnabledArgsForCall)]
	fake.autoCreateEnabledArgsForCall = append(fake.autoCreateEnabledArgsForCall, struct {
		arg1 context.Context
	}{arg1})
	stub := fake.AutoCreateEnabledStub
	fakeReturns := fake.autoCreateEnabledReturns
	fake.recordInvocation("AutoCreateEnabled", []interface{}{arg1})
	fake.autoCreateEnabledMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeRoomAllocator) AutoCreateEnabledCallCount() int {
	fake.autoCreateEnabledMutex.RLock()
	defer fake.autoCreateEnabledMutex.RUnlock()
	return len(fake.autoCreateEnabledArgsForCall)
}

func (fake *FakeRoomAllocator) AutoCreateEnabledCalls(stub func(context.Context) bool) {
	fake.autoCreateEnabledMutex.Lock()
	defer fake.autoCreateEnabledMutex.Unlock()
	fake.AutoCreateEnabledStub = stub
}

func (fake *FakeRoomAllocator) AutoCreateEnabledArgsForCall(i int) context.Context {
	fake.autoCreateEnabledMutex.RLock()
	defer fake.autoCreateEnabledMutex.RUnlock()
	argsForCall := fake.autoCreateEnabledArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeRoomAllocator) AutoCreateEnabledReturns(result1 bool) {
	fake.autoCreateEnabledMutex.Lock()
	defer fake.autoCreateEnabledMutex.Unlock()
	fake.AutoCreateEnabledStub = nil
	fake.autoCreateEnabledReturns = struct {
		result1 bool
	}{result1}
}

func (fake *FakeRoomAllocator) AutoCreateEnabledReturnsOnCall(i int, result1 bool) {
	fake.autoCreateEnabledMutex.Lock()
	defer fake.autoCreateEnabledMutex.Unlock()
	fake.AutoCreateEnabledStub = nil
	if fake.autoCreateEnabledReturnsOnCall == nil {
		fake.autoCreateEnabledReturnsOnCall = make(map[int]struct {
			result1 bool
		})
	}
	fake.autoCreateEnabledReturnsOnCall[i] = struct {
		result1 bool
	}{result1}
}

func (fake *FakeRoomAllocator) CreateRoom(arg1 context.Context, arg2 *conkit.CreateRoomRequest, arg3 bool) (*conkit.Room, *conkit.RoomInternal, bool, error) {
	fake.createRoomMutex.Lock()
	ret, specificReturn := fake.createRoomReturnsOnCall[len(fake.createRoomArgsForCall)]
	fake.createRoomArgsForCall = append(fake.createRoomArgsForCall, struct {
		arg1 context.Context
		arg2 *conkit.CreateRoomRequest
		arg3 bool
	}{arg1, arg2, arg3})
	stub := fake.CreateRoomStub
	fakeReturns := fake.createRoomReturns
	fake.recordInvocation("CreateRoom", []interface{}{arg1, arg2, arg3})
	fake.createRoomMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3)
	}
	if specificReturn {
		return ret.result1, ret.result2, ret.result3, ret.result4
	}
	return fakeReturns.result1, fakeReturns.result2, fakeReturns.result3, fakeReturns.result4
}

func (fake *FakeRoomAllocator) CreateRoomCallCount() int {
	fake.createRoomMutex.RLock()
	defer fake.createRoomMutex.RUnlock()
	return len(fake.createRoomArgsForCall)
}

func (fake *FakeRoomAllocator) CreateRoomCalls(stub func(context.Context, *conkit.CreateRoomRequest, bool) (*conkit.Room, *conkit.RoomInternal, bool, error)) {
	fake.createRoomMutex.Lock()
	defer fake.createRoomMutex.Unlock()
	fake.CreateRoomStub = stub
}

func (fake *FakeRoomAllocator) CreateRoomArgsForCall(i int) (context.Context, *conkit.CreateRoomRequest, bool) {
	fake.createRoomMutex.RLock()
	defer fake.createRoomMutex.RUnlock()
	argsForCall := fake.createRoomArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeRoomAllocator) CreateRoomReturns(result1 *conkit.Room, result2 *conkit.RoomInternal, result3 bool, result4 error) {
	fake.createRoomMutex.Lock()
	defer fake.createRoomMutex.Unlock()
	fake.CreateRoomStub = nil
	fake.createRoomReturns = struct {
		result1 *conkit.Room
		result2 *conkit.RoomInternal
		result3 bool
		result4 error
	}{result1, result2, result3, result4}
}

func (fake *FakeRoomAllocator) CreateRoomReturnsOnCall(i int, result1 *conkit.Room, result2 *conkit.RoomInternal, result3 bool, result4 error) {
	fake.createRoomMutex.Lock()
	defer fake.createRoomMutex.Unlock()
	fake.CreateRoomStub = nil
	if fake.createRoomReturnsOnCall == nil {
		fake.createRoomReturnsOnCall = make(map[int]struct {
			result1 *conkit.Room
			result2 *conkit.RoomInternal
			result3 bool
			result4 error
		})
	}
	fake.createRoomReturnsOnCall[i] = struct {
		result1 *conkit.Room
		result2 *conkit.RoomInternal
		result3 bool
		result4 error
	}{result1, result2, result3, result4}
}

func (fake *FakeRoomAllocator) SelectRoomNode(arg1 context.Context, arg2 conkit.RoomName, arg3 conkit.NodeID) error {
	fake.selectRoomNodeMutex.Lock()
	ret, specificReturn := fake.selectRoomNodeReturnsOnCall[len(fake.selectRoomNodeArgsForCall)]
	fake.selectRoomNodeArgsForCall = append(fake.selectRoomNodeArgsForCall, struct {
		arg1 context.Context
		arg2 conkit.RoomName
		arg3 conkit.NodeID
	}{arg1, arg2, arg3})
	stub := fake.SelectRoomNodeStub
	fakeReturns := fake.selectRoomNodeReturns
	fake.recordInvocation("SelectRoomNode", []interface{}{arg1, arg2, arg3})
	fake.selectRoomNodeMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeRoomAllocator) SelectRoomNodeCallCount() int {
	fake.selectRoomNodeMutex.RLock()
	defer fake.selectRoomNodeMutex.RUnlock()
	return len(fake.selectRoomNodeArgsForCall)
}

func (fake *FakeRoomAllocator) SelectRoomNodeCalls(stub func(context.Context, conkit.RoomName, conkit.NodeID) error) {
	fake.selectRoomNodeMutex.Lock()
	defer fake.selectRoomNodeMutex.Unlock()
	fake.SelectRoomNodeStub = stub
}

func (fake *FakeRoomAllocator) SelectRoomNodeArgsForCall(i int) (context.Context, conkit.RoomName, conkit.NodeID) {
	fake.selectRoomNodeMutex.RLock()
	defer fake.selectRoomNodeMutex.RUnlock()
	argsForCall := fake.selectRoomNodeArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeRoomAllocator) SelectRoomNodeReturns(result1 error) {
	fake.selectRoomNodeMutex.Lock()
	defer fake.selectRoomNodeMutex.Unlock()
	fake.SelectRoomNodeStub = nil
	fake.selectRoomNodeReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeRoomAllocator) SelectRoomNodeReturnsOnCall(i int, result1 error) {
	fake.selectRoomNodeMutex.Lock()
	defer fake.selectRoomNodeMutex.Unlock()
	fake.SelectRoomNodeStub = nil
	if fake.selectRoomNodeReturnsOnCall == nil {
		fake.selectRoomNodeReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.selectRoomNodeReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeRoomAllocator) ValidateCreateRoom(arg1 context.Context, arg2 conkit.RoomName) error {
	fake.validateCreateRoomMutex.Lock()
	ret, specificReturn := fake.validateCreateRoomReturnsOnCall[len(fake.validateCreateRoomArgsForCall)]
	fake.validateCreateRoomArgsForCall = append(fake.validateCreateRoomArgsForCall, struct {
		arg1 context.Context
		arg2 conkit.RoomName
	}{arg1, arg2})
	stub := fake.ValidateCreateRoomStub
	fakeReturns := fake.validateCreateRoomReturns
	fake.recordInvocation("ValidateCreateRoom", []interface{}{arg1, arg2})
	fake.validateCreateRoomMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeRoomAllocator) ValidateCreateRoomCallCount() int {
	fake.validateCreateRoomMutex.RLock()
	defer fake.validateCreateRoomMutex.RUnlock()
	return len(fake.validateCreateRoomArgsForCall)
}

func (fake *FakeRoomAllocator) ValidateCreateRoomCalls(stub func(context.Context, conkit.RoomName) error) {
	fake.validateCreateRoomMutex.Lock()
	defer fake.validateCreateRoomMutex.Unlock()
	fake.ValidateCreateRoomStub = stub
}

func (fake *FakeRoomAllocator) ValidateCreateRoomArgsForCall(i int) (context.Context, conkit.RoomName) {
	fake.validateCreateRoomMutex.RLock()
	defer fake.validateCreateRoomMutex.RUnlock()
	argsForCall := fake.validateCreateRoomArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeRoomAllocator) ValidateCreateRoomReturns(result1 error) {
	fake.validateCreateRoomMutex.Lock()
	defer fake.validateCreateRoomMutex.Unlock()
	fake.ValidateCreateRoomStub = nil
	fake.validateCreateRoomReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeRoomAllocator) ValidateCreateRoomReturnsOnCall(i int, result1 error) {
	fake.validateCreateRoomMutex.Lock()
	defer fake.validateCreateRoomMutex.Unlock()
	fake.ValidateCreateRoomStub = nil
	if fake.validateCreateRoomReturnsOnCall == nil {
		fake.validateCreateRoomReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.validateCreateRoomReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeRoomAllocator) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.autoCreateEnabledMutex.RLock()
	defer fake.autoCreateEnabledMutex.RUnlock()
	fake.createRoomMutex.RLock()
	defer fake.createRoomMutex.RUnlock()
	fake.selectRoomNodeMutex.RLock()
	defer fake.selectRoomNodeMutex.RUnlock()
	fake.validateCreateRoomMutex.RLock()
	defer fake.validateCreateRoomMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeRoomAllocator) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ service.RoomAllocator = new(FakeRoomAllocator)
