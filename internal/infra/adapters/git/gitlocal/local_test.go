package local

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var output = `
@@@PREFIX@@@HEAD -> masterorigin/masterorigin/HEAD@@@SUFFIX@@@2024-06-20T07:47:59+0000
@@@PREFIX@@@~~~v1.9.3@@@SUFFIX@@@2024-06-20T07:31:02+0000
@@@PREFIX@@@~~~v1.9.2@@@SUFFIX@@@2024-06-06T13:57:29+0000
@@@PREFIX@@@~~~v0.3.5~~~v0.3.4~~~v0.3.3~~~v0.3.2@@@SUFFIX@@@2024-01-04T14:26:16+0100
@@@PREFIX@@@~~~v0.3.1@@@SUFFIX@@@2024-01-04T12:00:59+0100
`

func TestDecode(t *testing.T) {
	adapter := NewAdapter(AdapterOptions{})
	tags, err := adapter.decode(output)
	assert.Nil(t, err)
	assert.Equal(t, 7, len(tags))
	assert.Equal(t, "v1.9.3", tags[0].Name)
	assert.Equal(t, "2024-06-20T07:31:02Z", tags[0].Time.Format(time.RFC3339))
	assert.Equal(t, "v1.9.2", tags[1].Name)
	assert.Equal(t, "v0.3.5", tags[2].Name)
	assert.Equal(t, "2024-01-04T14:26:16+01:00", tags[2].Time.Format(time.RFC3339))
	assert.Equal(t, "v0.3.4", tags[3].Name)
	assert.Equal(t, "2024-01-04T14:26:16+01:00", tags[3].Time.Format(time.RFC3339))
	assert.Equal(t, "v0.3.1", tags[6].Name)
}
