package application_test

import (
	"cf"
	"cf/api"
	. "cf/commands/application"
	"github.com/stretchr/testify/assert"
	"testhelpers"
	"testing"
)

func TestUnsetEnvRequirements(t *testing.T) {
	app := cf.Application{Name: "my-app", Guid: "my-app-guid"}
	appRepo := &testhelpers.FakeApplicationRepository{}
	args := []string{"my-app", "DATABASE_URL"}

	reqFactory := &testhelpers.FakeReqFactory{Application: app, LoginSuccess: true, TargetedSpaceSuccess: true}
	callUnsetEnv(args, reqFactory, appRepo)
	assert.True(t, testhelpers.CommandDidPassRequirements)

	reqFactory = &testhelpers.FakeReqFactory{Application: app, LoginSuccess: false, TargetedSpaceSuccess: true}
	callUnsetEnv(args, reqFactory, appRepo)
	assert.False(t, testhelpers.CommandDidPassRequirements)

	reqFactory = &testhelpers.FakeReqFactory{Application: app, LoginSuccess: true, TargetedSpaceSuccess: false}
	callUnsetEnv(args, reqFactory, appRepo)
	assert.False(t, testhelpers.CommandDidPassRequirements)
}

func TestUnsetEnvWhenApplicationExists(t *testing.T) {
	app := cf.Application{Name: "my-app", Guid: "my-app-guid", EnvironmentVars: map[string]string{"foo": "bar", "DATABASE_URL": "mysql://example.com/my-db"}}
	reqFactory := &testhelpers.FakeReqFactory{Application: app, LoginSuccess: true, TargetedSpaceSuccess: true}
	appRepo := &testhelpers.FakeApplicationRepository{}

	args := []string{"my-app", "DATABASE_URL"}
	ui := callUnsetEnv(args, reqFactory, appRepo)

	assert.Contains(t, ui.Outputs[0], "Removing env variable")
	assert.Contains(t, ui.Outputs[0], "my-app")
	assert.Contains(t, ui.Outputs[0], "DATABASE_URL")
	assert.Contains(t, ui.Outputs[1], "OK")

	assert.Equal(t, reqFactory.ApplicationName, "my-app")
	assert.Equal(t, appRepo.SetEnvApp, app)
	assert.Equal(t, appRepo.SetEnvVars, map[string]string{"foo": "bar"})
}

func TestUnsetEnvWhenUnsettingTheEnvFails(t *testing.T) {
	app := cf.Application{Name: "my-app", Guid: "my-app-guid", EnvironmentVars: map[string]string{"DATABASE_URL": "mysql://example.com/my-db"}}
	reqFactory := &testhelpers.FakeReqFactory{Application: app, LoginSuccess: true, TargetedSpaceSuccess: true}
	appRepo := &testhelpers.FakeApplicationRepository{
		FindByNameApp: app,
		SetEnvErr:     true,
	}

	args := []string{"does-not-exist", "DATABASE_URL"}
	ui := callUnsetEnv(args, reqFactory, appRepo)

	assert.Contains(t, ui.Outputs[0], "Removing env variable")
	assert.Contains(t, ui.Outputs[1], "FAILED")
	assert.Contains(t, ui.Outputs[2], "Failed setting env")
}

func TestUnsetEnvWhenEnvVarDoesNotExist(t *testing.T) {
	app := cf.Application{Name: "my-app", Guid: "my-app-guid"}
	reqFactory := &testhelpers.FakeReqFactory{Application: app, LoginSuccess: true, TargetedSpaceSuccess: true}
	appRepo := &testhelpers.FakeApplicationRepository{}

	args := []string{"my-app", "DATABASE_URL"}
	ui := callUnsetEnv(args, reqFactory, appRepo)

	assert.Equal(t, len(ui.Outputs), 3)
	assert.Contains(t, ui.Outputs[0], "Removing env variable")
	assert.Contains(t, ui.Outputs[1], "OK")
	assert.Contains(t, ui.Outputs[2], "DATABASE_URL")
	assert.Contains(t, ui.Outputs[2], "was not set.")
}

func TestUnsetEnvFailsWithUsage(t *testing.T) {
	app := cf.Application{Name: "my-app", Guid: "my-app-guid"}
	reqFactory := &testhelpers.FakeReqFactory{Application: app, LoginSuccess: true, TargetedSpaceSuccess: true}
	appRepo := &testhelpers.FakeApplicationRepository{FindByNameApp: app}

	args := []string{"my-app", "DATABASE_URL"}
	ui := callUnsetEnv(args, reqFactory, appRepo)
	assert.False(t, ui.FailedWithUsage)

	args = []string{"my-app"}
	ui = callUnsetEnv(args, reqFactory, appRepo)
	assert.True(t, ui.FailedWithUsage)

	args = []string{}
	ui = callUnsetEnv(args, reqFactory, appRepo)
	assert.True(t, ui.FailedWithUsage)
}

func callUnsetEnv(args []string, reqFactory *testhelpers.FakeReqFactory, appRepo api.ApplicationRepository) (ui *testhelpers.FakeUI) {
	ui = new(testhelpers.FakeUI)
	ctxt := testhelpers.NewContext("unset-env", args)

	cmd := NewUnsetEnv(ui, appRepo)
	testhelpers.RunCommand(cmd, ctxt, reqFactory)
	return
}
