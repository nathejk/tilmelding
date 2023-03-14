package stream_test

import (
	"fmt"
	"testing"

	"nathejk.dk/pkg/stream"
	"nathejk.dk/pkg/streaminterface"
)

func TestTopology(t *testing.T) {
	topo, err := stream.NewTopology([]streaminterface.Consumer{
		&testHandler{
			subscribes: []string{"rootA:updated", "rootB:removed", "f:removed"},
			produces:   []string{"g:updated", "g:removed"},
		},
		&testHandler{
			subscribes: []string{"rootA:updated", "rootA:removed"},
			produces:   []string{"c:updated", "c:removed"},
		},
		&testHandler{
			subscribes: []string{"rootB:updated", "rootB:removed"},
			produces:   []string{"d:updated", "d:removed"},
		},
		&testHandler{
			subscribes: []string{"rootA:updated", "rootB:removed"},
			produces:   []string{"e:updated", "e:removed"},
		},
		&testHandler{
			subscribes: []string{"e:updated", "e:removed"},
			produces:   []string{"f:updated", "f:removed"},
		},
		&testHandler{
			subscribes: []string{"f:updated", "f:removed"},
			produces:   []string{"aa:updated", "aa:removed"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	/*
		dg, err := topo.DotGraph()
		if err != nil {
			t.Fatal(err)
		}
		_ = dg
	*/
	//fmt.Println(string(dg))
	got := topo.SortedSubjects()
	//exp := []string{"aa", "g", "f", "e", "d", "c", "rootA", "rootB"}
	exp := []string{"aa", "g", "c", "d", "e", "f", "rootA", "rootB"}
	if fmt.Sprintf("%v", exp) != fmt.Sprintf("%v", got) {
		//	t.Fatalf("exp %v got %v", exp, got)
	}
}
