//go:build gometa

package main

import (
	"reflect"
	"testing"

	"go.mau.fi/mautrix-meta/pkg/messagix/table"
)

const gmPeer3 = int64(100000000000004)

func gmGroupOf(h *gometaHandler, members ...int64) {
	h.threadGroup[gmGroup] = true
	rows := make([]*table.LSAddParticipantIdToGroupThread, 0, len(members))
	for _, m := range members {
		rows = append(rows, &table.LSAddParticipantIdToGroupThread{ThreadKey: gmGroup, ContactId: m})
	}
	h.parseTable(&table.LSTable{LSAddParticipantIdToGroupThread: rows})
}

// ---- members ----

func TestGometaGroupMembersAreOnItsChatWithoutUs(t *testing.T) {
	h, r := newTestGometa(t)
	gmGroupOf(h, gmPeer2, gmSelf, gmPeer)
	if got, want := r.lists[itoa(gmGroup)], []int64{gmPeer, gmPeer2}; !reflect.DeepEqual(got, want) {
		t.Errorf("members %v, want %v", got, want)
	}
}

func TestGometaOneToOneThreadHasNoMemberList(t *testing.T) {
	h, r := newTestGometa(t)
	h.parseTable(&table.LSTable{LSAddParticipantIdToGroupThread: []*table.LSAddParticipantIdToGroupThread{
		{ThreadKey: gmPeer, ContactId: gmPeer}, {ThreadKey: gmPeer, ContactId: gmSelf},
	}})
	if len(r.lists) != 0 {
		t.Errorf("members %v", r.lists)
	}
}

func TestGometaRemovedMembersLeaveTheList(t *testing.T) {
	h, r := newTestGometa(t)
	gmGroupOf(h, gmPeer, gmPeer2, gmPeer3)
	h.parseTable(&table.LSTable{LSRemoveParticipantFromThread: []*table.LSRemoveParticipantFromThread{{ThreadKey: gmGroup, ParticipantId: gmPeer2}}})
	if got, want := r.lists[itoa(gmGroup)], []int64{gmPeer, gmPeer3}; !reflect.DeepEqual(got, want) {
		t.Errorf("after a removal %v, want %v", got, want)
	}
	// a full list again: the old one is dropped first, whatever order the table has them in
	h.parseTable(&table.LSTable{
		LSAddParticipantIdToGroupThread:  []*table.LSAddParticipantIdToGroupThread{{ThreadKey: gmGroup, ContactId: gmPeer2}},
		LSRemoveAllParticipantsForThread: []*table.LSRemoveAllParticipantsForThread{{ThreadKey: gmGroup}},
	})
	if got, want := r.lists[itoa(gmGroup)], []int64{gmPeer2}; !reflect.DeepEqual(got, want) {
		t.Errorf("after a new list %v, want %v", got, want)
	}
	h.parseTable(&table.LSTable{LSRemoveAllParticipantsForThread: []*table.LSRemoveAllParticipantsForThread{{ThreadKey: gmGroup}}})
	if got, ok := r.lists[itoa(gmGroup)]; !ok || len(got) != 0 {
		t.Errorf("after all left %v (%v)", got, ok)
	}
}

func TestGometaOpenedGroupGetsItsMembers(t *testing.T) {
	h, r := newTestGometa(t)
	gmGroupOf(h, gmPeer)
	r.lists = nil
	h.pushMembers(gmGroup)
	if got := r.lists[itoa(gmGroup)]; !reflect.DeepEqual(got, []int64{gmPeer}) {
		t.Errorf("members %v", got)
	}
}

func TestGometaGroupRenamedElsewhereIsNamedHere(t *testing.T) {
	h, r := newTestGometa(t)
	h.threadGroup[gmGroup] = true
	h.parseTable(&table.LSTable{LSSyncUpdateThreadName: []*table.LSSyncUpdateThreadName{
		{ThreadKey: gmGroup, ThreadName: "Friends"},
		{ThreadKey: gmGroup, ThreadName1: "Old friends"},
		{ThreadKey: gmPeer, ThreadName: "not a group"},
		{ThreadKey: gmGroup},
	}})
	want := []gmChat{{gmGroup, "Friends"}, {gmGroup, "Old friends"}}
	if !reflect.DeepEqual(r.chats, want) {
		t.Errorf("chats %+v, want %+v", r.chats, want)
	}
}

// ---- what the app may change ----
