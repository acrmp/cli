package requirements_test

import (
	. "cf/requirements"
	"github.com/stretchr/testify/assert"
	"testhelpers"
	"testing"
)

func TestValidAccessRequirement(t *testing.T) {
	ui := new(testhelpers.FakeUI)
	appRepo := &testhelpers.FakeApplicationRepository{
		FindByNameAuthErr: true,
	}

	req := NewValidAccessTokenRequirement(ui, appRepo)
	success := req.Execute()
	assert.False(t, success)
	assert.Contains(t, ui.Outputs[0], "Not logged in.")

	appRepo.FindByNameAuthErr = false

	req = NewValidAccessTokenRequirement(ui, appRepo)
	success = req.Execute()
	assert.True(t, success)
}
