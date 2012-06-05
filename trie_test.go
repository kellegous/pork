package pork

import "testing"

func expectMiss(t *testing.T, trie *Trie, k string) {
	_, _, found := trie.Get(k)
	if found {
		t.Errorf("expected NOT to find %s but did", k)
	}
}

func expectHit(t *testing.T, trie *Trie, k, p, v string) {
	rp, rv, found := trie.Get(k)
	if !found {
		t.Errorf("expected to find %s but did not", k)
	}

	if p != rp {
		t.Errorf("prefix: expected %s, got %s\n", p, rp)
	}

	value := rv.(string)
	if v != rv {
		t.Errorf("value: expected %s, got %s\n", v, value)
	}
}

func TestTrie(t *testing.T) {
	trie := NewTrie()
	trie.Put("k", "norton")
	trie.Put("kel", "naughton")

	expectHit(t, trie, "kellegous", "kel", "naughton")
	expectHit(t, trie, "kathy", "k", "norton")
	expectHit(t, trie, "kel", "kel", "naughton")
	expectHit(t, trie, "k", "k", "norton")
	expectMiss(t, trie, "foot")

	// Empty, catch all?
	trie.Put("", "bottom")
	expectHit(t, trie, "foo", "", "bottom")

	// Updates?
	trie.Put("", "new bottom")
	trie.Put("kel", "new naughton")
	expectHit(t, trie, "foo", "", "new bottom")
	expectHit(t, trie, "kelwelwoo", "kel", "new naughton")
}
