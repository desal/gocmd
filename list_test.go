package gocmd

import (
	"testing"

	"github.com/desal/cmd"
	"github.com/desal/richtext"
	"github.com/stretchr/testify/assert"
)

func TestList(t *testing.T) {
	output := cmd.NewStdOutput(richtext.Ansi())
	ctx := FromEnv(output)
	res, err := ctx.List("", "github.com/desal/...")
	assert.Nil(t, err)
	assert.Equal(t, "gocmd", res["github.com/desal/gocmd"]["Name"])
}
