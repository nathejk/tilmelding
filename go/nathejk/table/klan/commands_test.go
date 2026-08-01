package klan

import (
	"context"
	"testing"

	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/streamtest"
)

func newTestCommander() (*commander, *streamtest.SingleDomainPublisher) {
	pub := make(streamtest.SingleDomainPublisher, 16)
	return &commander{p: &pub}, &pub
}

func drain(pub *streamtest.SingleDomainPublisher) []stream.Message {
	var msgs []stream.Message
	for {
		m, ok := pub.Pop()
		if !ok {
			break
		}
		msgs = append(msgs, m)
	}
	return msgs
}

func TestAddMemberPublishesOneCreateEvent(t *testing.T) {
	c, pub := newTestCommander()
	id, err := c.AddMember(context.Background(), "team-1", Senior{Name: "Bo"})
	if err != nil {
		t.Fatalf("AddMember: %v", err)
	}
	if id == "" {
		t.Fatal("AddMember should issue a memberId")
	}
	msgs := drain(pub)
	if len(msgs) != 1 {
		t.Fatalf("want exactly 1 event, got %d", len(msgs))
	}
	if !msgs[0].Subject().Match("nathejk.*.senior.*.updated") {
		t.Errorf("unexpected subject %q", msgs[0].Subject().Subject())
	}
	var body struct {
		TeamID string `json:"teamId"`
	}
	if err := msgs[0].Body(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.TeamID == "" {
		t.Error("add event should carry teamId (create path)")
	}
}

func TestUpdateMemberPublishesOneUpdateEventWithoutCreate(t *testing.T) {
	c, pub := newTestCommander()
	if err := c.UpdateMember(context.Background(), "team-1", Senior{MemberID: "m-1", Name: "Bo"}); err != nil {
		t.Fatalf("UpdateMember: %v", err)
	}
	msgs := drain(pub)
	if len(msgs) != 1 {
		t.Fatalf("want exactly 1 event, got %d", len(msgs))
	}
	if !msgs[0].Subject().Match("nathejk.*.senior.*.updated") {
		t.Errorf("unexpected subject %q", msgs[0].Subject().Subject())
	}
	var body struct {
		TeamID string `json:"teamId"`
	}
	if err := msgs[0].Body(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.TeamID != "" {
		t.Errorf("update event must not carry teamId, got %q", body.TeamID)
	}
}

func TestDeleteMemberPublishesOneDeleteEvent(t *testing.T) {
	c, pub := newTestCommander()
	if err := c.DeleteMember(context.Background(), "team-1", "m-1"); err != nil {
		t.Fatalf("DeleteMember: %v", err)
	}
	msgs := drain(pub)
	if len(msgs) != 1 {
		t.Fatalf("want exactly 1 event, got %d", len(msgs))
	}
	if !msgs[0].Subject().Match("nathejk.*.senior.*.deleted") {
		t.Errorf("unexpected subject %q", msgs[0].Subject().Subject())
	}
}
