package pork

type Trie struct {
	children map[string]*Trie
	value    interface{}
}

func NewTrie() *Trie {
	return &Trie{map[string]*Trie{}, nil}
}

func (t *Trie) Put(prefix string, value interface{}) {
	for i, n := 0, len(prefix); i < n; i++ {
		k := prefix[i : i+1]
		c := t.children
		n, ok := c[k]
		if !ok {
			n = NewTrie()
			c[k] = n
		}
		t = n
	}
	t.value = value
}

func (t *Trie) Get(key string) (string, interface{}, bool) {
	depth, value := 0, t.value
	for i, n := 0, len(key); i < n; i++ {
		c, k := t.children, key[i:i+1]
		n, ok := c[k]
		if !ok {
			break
		}
		v := n.value
		if v != nil {
			value = v
			depth = i + 1
		}
		t = n
	}

	if value != nil {
		return key[0 : depth], value, true
	}
	return "", nil, false
}
