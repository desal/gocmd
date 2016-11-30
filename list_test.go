package gocmd_test

import (
	"testing"

	"github.com/desal/gocmd"
	"github.com/desal/richtext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestList(t *testing.T) {
	goPath, err := gocmd.EnvGoPath()
	require.Nil(t, err)
	ctx := gocmd.New(richtext.Test(t), goPath, "", "")
	res, err := ctx.List("", "github.com/desal/...")
	assert.Nil(t, err)
	assert.Equal(t, "gocmd", res["github.com/desal/gocmd"]["Name"])
}
