package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFirstNonEmptyString(t *testing.T) {
	assert.Equal(t, "a", FirstNonEmptyString("", "a"))
	assert.Equal(t, "a", FirstNonEmptyString("a", ""))
	assert.Equal(t, "a", FirstNonEmptyString("a", "b"))
	assert.Equal(t, "", FirstNonEmptyString("", ""))
}
