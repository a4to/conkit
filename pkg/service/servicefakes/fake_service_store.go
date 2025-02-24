// Code generated by counterfeiter. DO NOT EDIT.
package servicefakes

import (
	"context"
	"sync"

	"github.com/a4to/conkit-server/pkg/service"
	"github.com/a4to/protocol/conkit"
)

type FakeServiceStore struct {
	DeleteRoomStub        func(context.Context, conkit.RoomName) error
	deleteRoomMutex       sync.RWMutex
	deleteRoomArgsForCall []struct {
		arg1 context.Context
		arg2 conkit.RoomName
	}
	deleteRoomReturns struct {
		result1 error
	}
	deleteRoomReturnsOnCall map[int]struct {
		result1 error
	}
	ListParticipantsStub        func(context.Context, conkit.RoomName) ([]*conkit.ParticipantInfo, error)
	listParticipantsMutex       sync.RWMutex
	listParticipantsArgsForCall []struct {
		arg1 context.Context
		arg2 conkit.RoomName
	}
	listParticipantsReturns struct {
		result1 []*conkit.ParticipantInfo
		result2 error
	}
	listParticipantsReturnsOnCall map[int]struct {
		result1 []*conkit.ParticipantInfo
		result2 error
	}
	ListRoomsStub        func(context.Context, []conkit.RoomName) ([]*conkit.Room, error)
	listRoomsMutex       sync.RWMutex
	listRoomsArgsForCall []struct {
		arg1 context.Context
		arg2 []conkit.RoomName
	}
	listRoomsReturns struct {
		result1 []*conkit.Room
		result2 error
	}
	listRoomsReturnsOnCall map[int]struct {
		result1 []*conkit.Room
		result2 error
	}
	LoadParticipantStub        func(context.Context, conkit.RoomName, conkit.ParticipantIdentity) (*conkit.ParticipantInfo, error)
	loadParticipantMutex       sync.RWMutex
	loadParticipantArgsForCall []struct {
		arg1 context.Context
		arg2 conkit.RoomName
		arg3 conkit.ParticipantIdentity
	}
	loadParticipantReturns struct {
		result1 *conkit.ParticipantInfo
		result2 error
	}
	loadParticipantReturnsOnCall map[int]struct {
		result1 *conkit.ParticipantInfo
		result2 error
	}
	LoadRoomStub        func(context.Context, conkit.RoomName, bool) (*conkit.Room, *conkit.RoomInternal, error)
	loadRoomMutex       sync.RWMutex
	loadRoomArgsForCall []struct {
		arg1 context.Context
		arg2 conkit.RoomName
		arg3 bool
	}
	loadRoomReturns struct {
		result1 *conkit.Room
		result2 *conkit.RoomInternal
		result3 error
	}
	loadRoomReturnsOnCall map[int]struct {
		result1 *conkit.Room
		result2 *conkit.RoomInternal
		result3 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeServiceStore) DeleteRoom(arg1 context.Context, arg2 conkit.RoomName) error {
	fake.deleteRoomMutex.Lock()
	ret, specificReturn := fake.deleteRoomReturnsOnCall[len(fake.deleteRoomArgsForCall)]
	fake.deleteRoomArgsForCall = append(fake.deleteRoomArgsForCall, struct {
		arg1 context.Context
		arg2 conkit.RoomName
	}{arg1, arg2})
	stub := fake.DeleteRoomStub
	fakeReturns := fake.deleteRoomReturns
	fake.recordInvocation("DeleteRoom", []interface{}{arg1, arg2})
	fake.deleteRoomMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeServiceStore) DeleteRoomCallCount() int {
	fake.deleteRoomMutex.RLock()
	defer fake.deleteRoomMutex.RUnlock()
	return len(fake.deleteRoomArgsForCall)
}

func (fake *FakeServiceStore) DeleteRoomCalls(stub func(context.Context, conkit.RoomName) error) {
	fake.deleteRoomMutex.Lock()
	defer fake.deleteRoomMutex.Unlock()
	fake.DeleteRoomStub = stub
}

func (fake *FakeServiceStore) DeleteRoomArgsForCall(i int) (context.Context, conkit.RoomName) {
	fake.deleteRoomMutex.RLock()
	defer fake.deleteRoomMutex.RUnlock()
	argsForCall := fake.deleteRoomArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeServiceStore) DeleteRoomReturns(result1 error) {
	fake.deleteRoomMutex.Lock()
	defer fake.deleteRoomMutex.Unlock()
	fake.DeleteRoomStub = nil
	fake.deleteRoomReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeServiceStore) DeleteRoomReturnsOnCall(i int, result1 error) {
	fake.deleteRoomMutex.Lock()
	defer fake.deleteRoomMutex.Unlock()
	fake.DeleteRoomStub = nil
	if fake.deleteRoomReturnsOnCall == nil {
		fake.deleteRoomReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.deleteRoomReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeServiceStore) ListParticipants(arg1 context.Context, arg2 conkit.RoomName) ([]*conkit.ParticipantInfo, error) {
	fake.listParticipantsMutex.Lock()
	ret, specificReturn := fake.listParticipantsReturnsOnCall[len(fake.listParticipantsArgsForCall)]
	fake.listParticipantsArgsForCall = append(fake.listParticipantsArgsForCall, struct {
		arg1 context.Context
		arg2 conkit.RoomName
	}{arg1, arg2})
	stub := fake.ListParticipantsStub
	fakeReturns := fake.listParticipantsReturns
	fake.recordInvocation("ListParticipants", []interface{}{arg1, arg2})
	fake.listParticipantsMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeServiceStore) ListParticipantsCallCount() int {
	fake.listParticipantsMutex.RLock()
	defer fake.listParticipantsMutex.RUnlock()
	return len(fake.listParticipantsArgsForCall)
}

func (fake *FakeServiceStore) ListParticipantsCalls(stub func(context.Context, conkit.RoomName) ([]*conkit.ParticipantInfo, error)) {
	fake.listParticipantsMutex.Lock()
	defer fake.listParticipantsMutex.Unlock()
	fake.ListParticipantsStub = stub
}

func (fake *FakeServiceStore) ListParticipantsArgsForCall(i int) (context.Context, conkit.RoomName) {
	fake.listParticipantsMutex.RLock()
	defer fake.listParticipantsMutex.RUnlock()
	argsForCall := fake.listParticipantsArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeServiceStore) ListParticipantsReturns(result1 []*conkit.ParticipantInfo, result2 error) {
	fake.listParticipantsMutex.Lock()
	defer fake.listParticipantsMutex.Unlock()
	fake.ListParticipantsStub = nil
	fake.listParticipantsReturns = struct {
		result1 []*conkit.ParticipantInfo
		result2 error
	}{result1, result2}
}

func (fake *FakeServiceStore) ListParticipantsReturnsOnCall(i int, result1 []*conkit.ParticipantInfo, result2 error) {
	fake.listParticipantsMutex.Lock()
	defer fake.listParticipantsMutex.Unlock()
	fake.ListParticipantsStub = nil
	if fake.listParticipantsReturnsOnCall == nil {
		fake.listParticipantsReturnsOnCall = make(map[int]struct {
			result1 []*conkit.ParticipantInfo
			result2 error
		})
	}
	fake.listParticipantsReturnsOnCall[i] = struct {
		result1 []*conkit.ParticipantInfo
		result2 error
	}{result1, result2}
}

func (fake *FakeServiceStore) ListRooms(arg1 context.Context, arg2 []conkit.RoomName) ([]*conkit.Room, error) {
	var arg2Copy []conkit.RoomName
	if arg2 != nil {
		arg2Copy = make([]conkit.RoomName, len(arg2))
		copy(arg2Copy, arg2)
	}
	fake.listRoomsMutex.Lock()
	ret, specificReturn := fake.listRoomsReturnsOnCall[len(fake.listRoomsArgsForCall)]
	fake.listRoomsArgsForCall = append(fake.listRoomsArgsForCall, struct {
		arg1 context.Context
		arg2 []conkit.RoomName
	}{arg1, arg2Copy})
	stub := fake.ListRoomsStub
	fakeReturns := fake.listRoomsReturns
	fake.recordInvocation("ListRooms", []interface{}{arg1, arg2Copy})
	fake.listRoomsMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeServiceStore) ListRoomsCallCount() int {
	fake.listRoomsMutex.RLock()
	defer fake.listRoomsMutex.RUnlock()
	return len(fake.listRoomsArgsForCall)
}

func (fake *FakeServiceStore) ListRoomsCalls(stub func(context.Context, []conkit.RoomName) ([]*conkit.Room, error)) {
	fake.listRoomsMutex.Lock()
	defer fake.listRoomsMutex.Unlock()
	fake.ListRoomsStub = stub
}

func (fake *FakeServiceStore) ListRoomsArgsForCall(i int) (context.Context, []conkit.RoomName) {
	fake.listRoomsMutex.RLock()
	defer fake.listRoomsMutex.RUnlock()
	argsForCall := fake.listRoomsArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeServiceStore) ListRoomsReturns(result1 []*conkit.Room, result2 error) {
	fake.listRoomsMutex.Lock()
	defer fake.listRoomsMutex.Unlock()
	fake.ListRoomsStub = nil
	fake.listRoomsReturns = struct {
		result1 []*conkit.Room
		result2 error
	}{result1, result2}
}

func (fake *FakeServiceStore) ListRoomsReturnsOnCall(i int, result1 []*conkit.Room, result2 error) {
	fake.listRoomsMutex.Lock()
	defer fake.listRoomsMutex.Unlock()
	fake.ListRoomsStub = nil
	if fake.listRoomsReturnsOnCall == nil {
		fake.listRoomsReturnsOnCall = make(map[int]struct {
			result1 []*conkit.Room
			result2 error
		})
	}
	fake.listRoomsReturnsOnCall[i] = struct {
		result1 []*conkit.Room
		result2 error
	}{result1, result2}
}

func (fake *FakeServiceStore) LoadParticipant(arg1 context.Context, arg2 conkit.RoomName, arg3 conkit.ParticipantIdentity) (*conkit.ParticipantInfo, error) {
	fake.loadParticipantMutex.Lock()
	ret, specificReturn := fake.loadParticipantReturnsOnCall[len(fake.loadParticipantArgsForCall)]
	fake.loadParticipantArgsForCall = append(fake.loadParticipantArgsForCall, struct {
		arg1 context.Context
		arg2 conkit.RoomName
		arg3 conkit.ParticipantIdentity
	}{arg1, arg2, arg3})
	stub := fake.LoadParticipantStub
	fakeReturns := fake.loadParticipantReturns
	fake.recordInvocation("LoadParticipant", []interface{}{arg1, arg2, arg3})
	fake.loadParticipantMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeServiceStore) LoadParticipantCallCount() int {
	fake.loadParticipantMutex.RLock()
	defer fake.loadParticipantMutex.RUnlock()
	return len(fake.loadParticipantArgsForCall)
}

func (fake *FakeServiceStore) LoadParticipantCalls(stub func(context.Context, conkit.RoomName, conkit.ParticipantIdentity) (*conkit.ParticipantInfo, error)) {
	fake.loadParticipantMutex.Lock()
	defer fake.loadParticipantMutex.Unlock()
	fake.LoadParticipantStub = stub
}

func (fake *FakeServiceStore) LoadParticipantArgsForCall(i int) (context.Context, conkit.RoomName, conkit.ParticipantIdentity) {
	fake.loadParticipantMutex.RLock()
	defer fake.loadParticipantMutex.RUnlock()
	argsForCall := fake.loadParticipantArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeServiceStore) LoadParticipantReturns(result1 *conkit.ParticipantInfo, result2 error) {
	fake.loadParticipantMutex.Lock()
	defer fake.loadParticipantMutex.Unlock()
	fake.LoadParticipantStub = nil
	fake.loadParticipantReturns = struct {
		result1 *conkit.ParticipantInfo
		result2 error
	}{result1, result2}
}

func (fake *FakeServiceStore) LoadParticipantReturnsOnCall(i int, result1 *conkit.ParticipantInfo, result2 error) {
	fake.loadParticipantMutex.Lock()
	defer fake.loadParticipantMutex.Unlock()
	fake.LoadParticipantStub = nil
	if fake.loadParticipantReturnsOnCall == nil {
		fake.loadParticipantReturnsOnCall = make(map[int]struct {
			result1 *conkit.ParticipantInfo
			result2 error
		})
	}
	fake.loadParticipantReturnsOnCall[i] = struct {
		result1 *conkit.ParticipantInfo
		result2 error
	}{result1, result2}
}

func (fake *FakeServiceStore) LoadRoom(arg1 context.Context, arg2 conkit.RoomName, arg3 bool) (*conkit.Room, *conkit.RoomInternal, error) {
	fake.loadRoomMutex.Lock()
	ret, specificReturn := fake.loadRoomReturnsOnCall[len(fake.loadRoomArgsForCall)]
	fake.loadRoomArgsForCall = append(fake.loadRoomArgsForCall, struct {
		arg1 context.Context
		arg2 conkit.RoomName
		arg3 bool
	}{arg1, arg2, arg3})
	stub := fake.LoadRoomStub
	fakeReturns := fake.loadRoomReturns
	fake.recordInvocation("LoadRoom", []interface{}{arg1, arg2, arg3})
	fake.loadRoomMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3)
	}
	if specificReturn {
		return ret.result1, ret.result2, ret.result3
	}
	return fakeReturns.result1, fakeReturns.result2, fakeReturns.result3
}

func (fake *FakeServiceStore) LoadRoomCallCount() int {
	fake.loadRoomMutex.RLock()
	defer fake.loadRoomMutex.RUnlock()
	return len(fake.loadRoomArgsForCall)
}

func (fake *FakeServiceStore) LoadRoomCalls(stub func(context.Context, conkit.RoomName, bool) (*conkit.Room, *conkit.RoomInternal, error)) {
	fake.loadRoomMutex.Lock()
	defer fake.loadRoomMutex.Unlock()
	fake.LoadRoomStub = stub
}

func (fake *FakeServiceStore) LoadRoomArgsForCall(i int) (context.Context, conkit.RoomName, bool) {
	fake.loadRoomMutex.RLock()
	defer fake.loadRoomMutex.RUnlock()
	argsForCall := fake.loadRoomArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeServiceStore) LoadRoomReturns(result1 *conkit.Room, result2 *conkit.RoomInternal, result3 error) {
	fake.loadRoomMutex.Lock()
	defer fake.loadRoomMutex.Unlock()
	fake.LoadRoomStub = nil
	fake.loadRoomReturns = struct {
		result1 *conkit.Room
		result2 *conkit.RoomInternal
		result3 error
	}{result1, result2, result3}
}

func (fake *FakeServiceStore) LoadRoomReturnsOnCall(i int, result1 *conkit.Room, result2 *conkit.RoomInternal, result3 error) {
	fake.loadRoomMutex.Lock()
	defer fake.loadRoomMutex.Unlock()
	fake.LoadRoomStub = nil
	if fake.loadRoomReturnsOnCall == nil {
		fake.loadRoomReturnsOnCall = make(map[int]struct {
			result1 *conkit.Room
			result2 *conkit.RoomInternal
			result3 error
		})
	}
	fake.loadRoomReturnsOnCall[i] = struct {
		result1 *conkit.Room
		result2 *conkit.RoomInternal
		result3 error
	}{result1, result2, result3}
}

func (fake *FakeServiceStore) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.deleteRoomMutex.RLock()
	defer fake.deleteRoomMutex.RUnlock()
	fake.listParticipantsMutex.RLock()
	defer fake.listParticipantsMutex.RUnlock()
	fake.listRoomsMutex.RLock()
	defer fake.listRoomsMutex.RUnlock()
	fake.loadParticipantMutex.RLock()
	defer fake.loadParticipantMutex.RUnlock()
	fake.loadRoomMutex.RLock()
	defer fake.loadRoomMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeServiceStore) recordInvocation(key string, args []interface{}) {
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

var _ service.ServiceStore = new(FakeServiceStore)
