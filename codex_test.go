package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestCodexDecodeAndMerge(t *testing.T) {
	for _, bad := range []string{`{`, `null`, `{"version":2}`, `{"items":{"-1":{}}}`, `{"enemies":{"1":{"defeated":-1}}}`} {
		if _, err := decodeCodex([]byte(bad)); err == nil {
			t.Errorf("accepted %s", bad)
		}
	}
	c, err := decodeCodex([]byte(`{"items":{"5":{"identified":true}}}`))
	if err != nil || !c.Items[5].Seen {
		t.Fatal(c, err)
	}
	c.Enemies[0] = CodexRecord{Seen: true, Defeated: 2}
	data, _ := json.Marshal(c)
	back, err := decodeCodex(data)
	if err != nil || !reflect.DeepEqual(c, back) {
		t.Fatal(err, back)
	}
	d := newCodexData()
	d.Enemies[0] = CodexRecord{Deaths: 3}
	merged := mergeCodex(c, d)
	if merged.Enemies[0].Defeated != 2 || merged.Enemies[0].Deaths != 3 || !reflect.DeepEqual(merged, mergeCodex(merged, merged)) {
		t.Fatal(merged)
	}
	n, m, p := codexDiscovery([]int{0, 2}, merged.Enemies)
	if n != 1 || m != 2 || p != 50 {
		t.Fatal(n, m, p)
	}
	n, m, p = codexDiscovery(nil, nil)
	if n != 0 || m != 0 || p != 0 {
		t.Fatal(n, m, p)
	}
}
