package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFirstFiveChars(t *testing.T) {
	assert.Equal(t, "ASDFG", FirstFiveChars("asdfghj"))
	assert.Equal(t, "ASDFG", FirstFiveChars("asdfg"))
	assert.Equal(t, "ASDF", FirstFiveChars("asdf"))
	assert.Equal(t, "ASD", FirstFiveChars("asd"))
	assert.Equal(t, "AS", FirstFiveChars("as"))
	assert.Equal(t, "A", FirstFiveChars("a"))
	assert.Equal(t, "", FirstFiveChars(""))
}

func TestFirstChar(t *testing.T) {
	assert.Equal(t, "A", FirstChar("asdfghj"))
	assert.Equal(t, "A", FirstChar("a"))
	assert.Equal(t, "", FirstChar(""))
}
