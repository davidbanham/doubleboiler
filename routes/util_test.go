package routes

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNextFlow(t *testing.T) {
	t.Parallel()

	// Respect the default
	defaultURL := "/users/hai?aoo=bar"
	assert.Equal(t, defaultURL, nextFlow(defaultURL, url.Values{}))

	// Add in values
	assert.Equal(t, defaultURL+"&flow=quux", nextFlow(defaultURL, url.Values{
		"flow": []string{"quux"},
	}))

	// Overwrite values
	assert.Equal(t, defaultURL+"&flow=yeah", nextFlow(defaultURL+"&flow=nah", url.Values{
		"flow": []string{"yeah"},
	}))

	// Respect root default
	assert.Equal(t, "", nextFlow("", url.Values{}))

	// Override root default with a next
	assert.Equal(t, "users/lolanid?flow=heapsgood", nextFlow("", url.Values{
		"next": []string{"users/lolanid"},
		"flow": []string{"heapsgood"},
	}))

	// Pass root default with a flow
	assert.Equal(t, "?flow=heapsgood", nextFlow("", url.Values{
		"flow": []string{"heapsgood"},
	}))

	// Never end up in a login loop
	assert.Equal(t, "yay", nextFlow("yay", url.Values{
		"next": []string{"/login"},
	}))

	assert.Equal(t, "yay", nextFlow("yay", url.Values{
		"next": []string{"login"},
	}))
}
