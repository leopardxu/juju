// Copyright 2012, 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package jujuc_test

import (
	"io/ioutil"
	. "launchpad.net/gocheck"
	"launchpad.net/juju-core/cmd"
	"launchpad.net/juju-core/testing"
	"launchpad.net/juju-core/worker/uniter/jujuc"
	"path/filepath"
)

type ConfigGetSuite struct {
	ContextSuite
}

var _ = Suite(&ConfigGetSuite{})

var (
	configGetYamlMap    = "monsters: false\nspline-reticulation: 45\ntitle: My Title\nusername: admin001\n"
	configGetYamlMapAll = "empty: null\nmonsters: false\nspline-reticulation: 45\ntitle: My Title\nusername: admin001\n"
	configGetJsonMap    = `{"monsters":false,"spline-reticulation":45,"title":"My Title","username":"admin001"}` + "\n"
	configGetJsonMapAll = `{"empty":null,"monsters":false,"spline-reticulation":45,"title":"My Title","username":"admin001"}` + "\n"
)

var configGetTests = []struct {
	args []string
	out  string
}{
	{[]string{"monsters"}, "False\n"},
	{[]string{"--format", "yaml", "monsters"}, "false\n"},
	{[]string{"--format", "json", "monsters"}, "false\n"},
	{[]string{"spline-reticulation"}, "45\n"},
	{[]string{"--format", "yaml", "spline-reticulation"}, "45\n"},
	{[]string{"--format", "json", "spline-reticulation"}, "45\n"},
	{[]string{"missing"}, ""},
	{[]string{"--format", "yaml", "missing"}, ""},
	{[]string{"--format", "json", "missing"}, "null\n"},
	{nil, configGetYamlMap},
	{[]string{"--format", "yaml"}, configGetYamlMap},
	{[]string{"--format", "json"}, configGetJsonMap},
	{[]string{"--all", "--format", "yaml"}, configGetYamlMapAll},
	{[]string{"--all", "--format", "json"}, configGetJsonMapAll},
}

func (s *ConfigGetSuite) TestOutputFormat(c *C) {
	for i, t := range configGetTests {
		c.Logf("test %d: %#v", i, t.args)
		hctx := s.GetHookContext(c, -1, "")
		com, err := jujuc.NewCommand(hctx, "config-get")
		c.Assert(err, IsNil)
		ctx := testing.Context(c)
		code := cmd.Main(com, ctx, t.args)
		c.Assert(code, Equals, 0)
		c.Assert(bufferString(ctx.Stderr), Equals, "")
		c.Assert(bufferString(ctx.Stdout), Matches, t.out)
	}
}

func (s *ConfigGetSuite) TestHelp(c *C) {
	hctx := s.GetHookContext(c, -1, "")
	com, err := jujuc.NewCommand(hctx, "config-get")
	c.Assert(err, IsNil)
	ctx := testing.Context(c)
	code := cmd.Main(com, ctx, []string{"--help"})
	c.Assert(code, Equals, 0)
	c.Assert(bufferString(ctx.Stdout), Equals, `usage: config-get [options] [<key>]
purpose: print service configuration

options:
-a, --all  (= false)
    write also keys without values
--format  (= smart)
    specify output format (json|smart|yaml)
-o, --output (= "")
    specify an output file

If a key is given, only the value for that key will be printed.
`)
	c.Assert(bufferString(ctx.Stderr), Equals, "")
}

func (s *ConfigGetSuite) TestOutputPath(c *C) {
	hctx := s.GetHookContext(c, -1, "")
	com, err := jujuc.NewCommand(hctx, "config-get")
	c.Assert(err, IsNil)
	ctx := testing.Context(c)
	code := cmd.Main(com, ctx, []string{"--output", "some-file", "monsters"})
	c.Assert(code, Equals, 0)
	c.Assert(bufferString(ctx.Stderr), Equals, "")
	c.Assert(bufferString(ctx.Stdout), Equals, "")
	content, err := ioutil.ReadFile(filepath.Join(ctx.Dir, "some-file"))
	c.Assert(err, IsNil)
	c.Assert(string(content), Equals, "False\n")
}

func (s *ConfigGetSuite) TestUnknownArg(c *C) {
	hctx := s.GetHookContext(c, -1, "")
	com, err := jujuc.NewCommand(hctx, "config-get")
	c.Assert(err, IsNil)
	testing.TestInit(c, com, []string{"multiple", "keys"}, `unrecognized args: \["keys"\]`)
}
