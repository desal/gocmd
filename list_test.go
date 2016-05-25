package gocmd

import (
	"testing"

	"github.com/desal/cmd"
	"github.com/stretchr/testify/assert"
)

func TestList(t *testing.T) {
	output := cmd.NewTestOutput(t)
	ctx := New(output, FromEnv(output))
	res, err := ctx.List("", "github.com/desal/...")
	assert.Nil(t, err)
	assert.Equal(t, "gocmd", res["github.com/desal/gocmd"]["Name"])
}
