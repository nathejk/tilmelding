package stream

import (
	"nathejk.dk/pkg/streaminterface"
)

type Topology interface {
	SortedSubjects() []string
	RootSubject(string) bool
}

type topology struct {
	consumed map[string]bool
	produced map[string]bool
}

func NewTopology(consumers []streaminterface.Consumer) (*topology, error) {
	t := &topology{consumed: map[string]bool{}, produced: map[string]bool{}}
	for _, c := range consumers {
		for _, s := range c.Consumes() {
			t.consumed[s.Domain()] = true
		}
		if p, ok := c.(streaminterface.Producer); ok {
			for _, s := range p.Produces() {
				t.produced[s.Domain()] = true
			}
		}
	}
	return t, nil
}

// We only need to sort subjects
//
// 1) create a nodes of all our subjects.
// 2) create edges by iterating each handlers subjects and their publishing
//    subjects.
// 3) iterate the nodes in reverse sorted order, subscribing handlers.
//
// All this ensures that we don't need any additional synchronization while
// setting up our topology of streams.
//
//    s1  s2
//    ^    ^
//   /      \
//  h1       h2
//   \
//    ˇ
//    s2
func (t *topology) SortedSubjects() []string {
	sorted := []string{}
	for s := range t.produced {
		if !t.consumed[s] {
			sorted = append(sorted, s)
		}
	}
	for s := range t.produced {
		if t.consumed[s] {
			sorted = append(sorted, s)
		}
	}
	for s := range t.consumed {
		if !t.produced[s] {
			sorted = append(sorted, s)
		}
	}
	return sorted
}
func (t *topology) RootSubject(subj string) bool {
	return t.consumed[subj] && !t.produced[subj]
}
