// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package cloud

import (
	"github.com/juju/cmd"
	"github.com/juju/errors"
	"github.com/juju/gnuflag"
	"gopkg.in/juju/names.v2"

	apicloud "github.com/juju/juju/api/cloud"
	jujucloud "github.com/juju/juju/cloud"
	"github.com/juju/juju/cmd/juju/common"
	"github.com/juju/juju/cmd/modelcmd"
)

var usageUpdateCredentialSummary = `
Updates a controller credential for a cloud.`[1:]

var usageUpdateCredentialDetails = `
Controller credentials are used for model operations and manipulations.
Since it is common to have long-running models, it is also common to 
have cloud credentials become invalid during models' lifetime.
When this happens, a user must update the controller credential that 
a model was created with to the new and valid details.

This command allows to update an existing, already-stored, named,
cloud-specific controller credential.

NOTE: 
This is the only command that will allow you to manipulate 
a controller credential. 
All other credential related commands, such as 
` + "`add-credential`" + `, ` + "`remove-credential`" + ` and  ` + "`credentials`" + ` 
deal with credentials stored locally on the client not on the controller.

Examples:
    juju update-credential aws mysecrets

See also: 
    add-credential
    credentials`[1:]

type updateCredentialCommand struct {
	modelcmd.ControllerCommandBase

	api credentialAPI

	cloud      string
	credential string
}

// NewUpdateCredentialCommand returns a command to update credential details.
func NewUpdateCredentialCommand() cmd.Command {
	return modelcmd.WrapController(&updateCredentialCommand{})
}

// Init implements Command.Init.
func (c *updateCredentialCommand) Init(args []string) error {
	if len(args) < 2 {
		return errors.New("Usage: juju update-credential <cloud-name> <credential-name>")
	}
	c.cloud = args[0]
	c.credential = args[1]
	return cmd.CheckEmpty(args[2:])
}

// Info implements Command.Info
func (c *updateCredentialCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "update-credential",
		Args:    "<cloud-name> <credential-name>",
		Purpose: usageUpdateCredentialSummary,
		Doc:     usageUpdateCredentialDetails,
	}
}

// SetFlags implements Command.SetFlags.
func (c *updateCredentialCommand) SetFlags(f *gnuflag.FlagSet) {
	c.ControllerCommandBase.SetFlags(f)
	f.StringVar(&c.credential, "credential", "", "Name of credential to update")
}

type credentialAPI interface {
	UpdateCredential(tag names.CloudCredentialTag, credential jujucloud.Credential) error
	Close() error
}

func (c *updateCredentialCommand) getAPI() (credentialAPI, error) {
	if c.api != nil {
		return c.api, nil
	}
	api, err := c.NewAPIRoot()
	if err != nil {
		return nil, errors.Annotate(err, "opening API connection")
	}
	return apicloud.NewClient(api), nil
}

// Run implements Command.Run
func (c *updateCredentialCommand) Run(ctx *cmd.Context) error {
	cred, err := c.ClientStore().CredentialForCloud(c.cloud)
	if errors.IsNotFound(err) {
		ctx.Infof("No credentials exist for cloud %q", c.cloud)
		return nil
	} else if err != nil {
		return err
	}
	credToUpdate, ok := cred.AuthCredentials[c.credential]
	if !ok {
		ctx.Infof("No credential called %q exists for cloud %q", c.credential, c.cloud)
		return nil
	}

	accountDetails, err := c.CurrentAccountDetails()
	if err != nil {
		return errors.Trace(err)
	}
	credentialTag, err := common.ResolveCloudCredentialTag(
		names.NewUserTag(accountDetails.User), names.NewCloudTag(c.cloud), c.credential,
	)

	client, err := c.getAPI()
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.UpdateCredential(credentialTag, credToUpdate); err != nil {
		return err
	}
	ctx.Infof("Updated credential %q for user %q on cloud %q.", c.credential, accountDetails.User, c.cloud)
	return nil
}
