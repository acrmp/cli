package organization

import (
	"cf"
	"cf/api"
	"cf/requirements"
	"cf/terminal"
	"errors"
	"github.com/codegangsta/cli"
)

type CreateOrg struct {
	ui      terminal.UI
	orgRepo api.OrganizationRepository
}

func NewCreateOrg(ui terminal.UI, orgRepo api.OrganizationRepository) (cmd CreateOrg) {
	cmd.ui = ui
	cmd.orgRepo = orgRepo
	return
}

func (cmd CreateOrg) GetRequirements(reqFactory requirements.Factory, c *cli.Context) (reqs []requirements.Requirement, err error) {
	if len(c.Args()) != 1 {
		err = errors.New("Incorrect Usage")
		cmd.ui.FailWithUsage(c, "create-org")
		return
	}

	reqs = []requirements.Requirement{
		reqFactory.NewLoginRequirement(),
	}
	return
}

func (cmd CreateOrg) Run(c *cli.Context) {
	name := c.Args()[0]

	cmd.ui.Say("Creating org %s...", terminal.EntityNameColor(name))
	apiResponse := cmd.orgRepo.Create(name)
	if apiResponse.IsNotSuccessful() {
		if apiResponse.ErrorCode == api.ORG_EXISTS {
			cmd.ui.Ok()
			cmd.ui.Warn("Org %s already exists", name)
			return
		}

		cmd.ui.Failed(apiResponse.Message)
		return
	}

	cmd.ui.Ok()
	cmd.ui.Say("\nTIP: Use '%s' to target new org", terminal.CommandColor(cf.Name+" target -o "+name))
}
