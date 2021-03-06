package route_test

import (
	"cf"
	. "cf/commands/route"
	"github.com/codegangsta/cli"
	"github.com/stretchr/testify/assert"
	"testhelpers"
	"testing"
)

func TestRouteMapperFailsWithUsage(t *testing.T) {
	reqFactory := &testhelpers.FakeReqFactory{}
	routeRepo := &testhelpers.FakeRouteRepository{}

	fakeUI := callRouteMapper([]string{}, reqFactory, routeRepo, true)
	assert.True(t, fakeUI.FailedWithUsage)

	fakeUI = callRouteMapper([]string{"foo"}, reqFactory, routeRepo, true)
	assert.True(t, fakeUI.FailedWithUsage)

	fakeUI = callRouteMapper([]string{"foo", "bar"}, reqFactory, routeRepo, true)
	assert.False(t, fakeUI.FailedWithUsage)
}

func TestRouteMapperRequirements(t *testing.T) {
	routeRepo := &testhelpers.FakeRouteRepository{}
	reqFactory := &testhelpers.FakeReqFactory{LoginSuccess: true}

	callRouteMapper([]string{"-n", "my-host", "my-app", "my-domain.com"}, reqFactory, routeRepo, true)
	assert.True(t, testhelpers.CommandDidPassRequirements)
	assert.Equal(t, reqFactory.ApplicationName, "my-app")
	assert.Equal(t, reqFactory.RouteHost, "my-host")
	assert.Equal(t, reqFactory.RouteDomain, "my-domain.com")
}

func TestRouteMapperWhenBinding(t *testing.T) {
	route := cf.Route{
		Guid:   "my-route-guid",
		Host:   "foo",
		Domain: cf.Domain{Guid: "my-domain-guid", Name: "example.com"},
	}
	app := cf.Application{Guid: "my-app-guid", Name: "my-app"}

	routeRepo := &testhelpers.FakeRouteRepository{}
	reqFactory := &testhelpers.FakeReqFactory{LoginSuccess: true, Route: route, Application: app}

	ui := callRouteMapper([]string{"-n", "my-host", "my-app", "my-domain.com"}, reqFactory, routeRepo, true)

	assert.Contains(t, ui.Outputs[0], "Adding url route")
	assert.Contains(t, ui.Outputs[0], "foo.example.com")
	assert.Contains(t, ui.Outputs[0], "my-app")

	assert.Equal(t, route, routeRepo.BoundRoute)
	assert.Equal(t, app, routeRepo.BoundApp)

	assert.Contains(t, ui.Outputs[1], "OK")
}

func TestRouteMapperWhenUnbinding(t *testing.T) {
	route := cf.Route{
		Guid:   "my-route-guid",
		Host:   "foo",
		Domain: cf.Domain{Guid: "my-domain-guid", Name: "example.com"},
	}
	app := cf.Application{Guid: "my-app-guid", Name: "my-app"}

	routeRepo := &testhelpers.FakeRouteRepository{}
	reqFactory := &testhelpers.FakeReqFactory{LoginSuccess: true, Route: route, Application: app}

	ui := callRouteMapper([]string{"-n", "my-host", "my-app", "my-domain.com"}, reqFactory, routeRepo, false)

	assert.Contains(t, ui.Outputs[0], "Removing url route")
	assert.Contains(t, ui.Outputs[0], "foo.example.com")
	assert.Contains(t, ui.Outputs[0], "my-app")

	assert.Equal(t, route, routeRepo.UnboundRoute)
	assert.Equal(t, app, routeRepo.UnboundApp)

	assert.Contains(t, ui.Outputs[1], "OK")
}

func callRouteMapper(args []string, reqFactory *testhelpers.FakeReqFactory, routeRepo *testhelpers.FakeRouteRepository, bind bool) (ui *testhelpers.FakeUI) {
	ui = new(testhelpers.FakeUI)
	var ctxt *cli.Context
	if bind {
		ctxt = testhelpers.NewContext("map-route", args)
	} else {
		ctxt = testhelpers.NewContext("unmap-route", args)
	}

	cmd := NewRouteMapper(ui, routeRepo, bind)
	testhelpers.RunCommand(cmd, ctxt, reqFactory)
	return
}
