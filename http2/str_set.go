package http2

type StringSet struct {
	inner map[string]struct{}
}

func NewStringSet() *StringSet {
	set := StringSet{}
	set.inner = make(map[string]struct{})
	return &set
}

func (set *StringSet) Add(str string) {
	set.inner[str] = struct{}{}
}

func (set *StringSet) Has(str string) bool {
	_, ok := set.inner[str]
	return ok
}

func (set *StringSet) Remove(str string) {
	delete(set.inner, str)
}
