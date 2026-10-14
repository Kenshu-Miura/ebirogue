package main

import (
	"reflect"
	"testing"
)

func TestSortedTemplateIDsHandlesGaps(t *testing.T) {
	templates := map[int]string{7: "c", 0: "a", 3: "boss", 2: "b"}
	got := sortedTemplateIDs(templates, func(name string) bool { return name != "boss" })
	if want := []int{0, 2, 7}; !reflect.DeepEqual(got, want) {
		t.Fatalf("sortedTemplateIDs = %v, want %v", got, want)
	}
	if all := sortedTemplateIDs(templates, nil); !reflect.DeepEqual(all, []int{0, 2, 3, 7}) {
		t.Fatalf("sortedTemplateIDs(nil) = %v", all)
	}
}

func TestPickTemplateIDOnlyReturnsListedIDs(t *testing.T) {
	ids := []int{0, 2, 7}
	for i := range ids {
		got := pickTemplateID(ids, func(n int) int {
			if n != len(ids) {
				t.Fatalf("intn called with %d, want %d", n, len(ids))
			}
			return i
		}, -1)
		if got != ids[i] {
			t.Fatalf("pickTemplateID(%d) = %d, want %d", i, got, ids[i])
		}
	}
	if got := pickTemplateID(nil, func(int) int { return 0 }, 5); got != 5 {
		t.Fatalf("empty ids should return fallback, got %d", got)
	}
}

func TestWeightedTemplateIDs(t *testing.T) {
	got := weightedTemplateIDs([]int{1, 2, 3}, map[int]int{2: 3, 3: 0})
	want := []int{1, 2, 2, 2}
	if len(got) != len(want) {
		t.Fatalf("pool = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("pool = %v, want %v", got, want)
		}
	}
}
