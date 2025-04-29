package util

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestListFilesRecursively(t *testing.T) {
	set := NewStringSet()
	err := ListFilesRecursively("./testdata", set)
	assert.Nil(t, err)
	t.Logf("set:%v", set.ToArray())
	assert.Equal(t, 4, set.Size())
}
