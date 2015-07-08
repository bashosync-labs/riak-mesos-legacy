package metadata_manager

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStub(t *testing.T) {
	assert.True(t, true, "This is good. Canary test passing")
}
func TestNS(t *testing.T) {
	assert := assert.New(t)
	bns := baseNamespace{}
	namespace := makeSubSpace(makeSubSpace(makeSubSpace(bns, "riak"), "frameworks"), "fakeFramework")

	assert.Equal([]string{"", "riak", "frameworks", "fakeFramework"}, namespace.GetComponents())
}
