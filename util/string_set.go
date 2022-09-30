package util

type StringSet struct {
	internal map[string]int
}

func NewStringSet() *StringSet {
	return &StringSet{internal: make(map[string]int)}
}

func (set *StringSet) Add(str string) {
	set.internal[str] = 1
}

func (set *StringSet) AddAll(itemSlice []string) {
	for _, item := range itemSlice {
		set.internal[item] = 1
	}
}

func (set *StringSet) Has(str string) bool {
	_, ok := set.internal[str]
	return ok
}

func (set *StringSet) Remove(str string) {
	delete(set.internal, str)
}

func (set *StringSet) RemoveAll(other *StringSet) {
	for _, item := range other.ToArray() {
		delete(set.internal, item)
	}
}

func (set *StringSet) ToArray() []string {
	res := make([]string, len(set.internal))
	i := 0
	for key := range set.internal {
		res[i] = key
		i++
	}
	return res
}

func (set *StringSet) Size() int {
	return len(set.internal)
}

func (set *StringSet) Intersection(set2 *StringSet) *StringSet {
	result := NewStringSet()

	if set.Size() > set2.Size() {
		set, set2 = set2, set
	}

	for key := range set.internal {
		if _, ok := set2.internal[key]; ok {
			result.Add(key)
		}
	}
	return result
}

func (set *StringSet) Clone() *StringSet {
	result := NewStringSet()
	result.AddAll(set.ToArray())
	return result
}
