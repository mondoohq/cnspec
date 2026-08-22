// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package forms

// Infrastructure as Code: ansible, bicep, cloudformation and terraform.
//
// All four are path-shaped -- each takes a file or a directory and no
// credential at all -- so the useful form is the one that names the path in the
// connector's own vocabulary. "PATH" is the label these specs exist to replace:
// the derived slot spells the usage string verbatim, so an uncurated bicep asks
// for "PATH" where it means "a .bicep file, a directory of them, or an ARM
// template".
//
// terraform is the one that also has to say *what shape* of Terraform is being
// read, because HCL source, a plan and a state file are three different things
// behind one connector word.

func init() {
	// The three IaC readers below take one path and declare no flags. Each is
	// its own registration rather than a shared spec because the only thing
	// each one carries is the sentence describing its own argument, and a
	// shared spec would have to drop exactly that.

	registerSpec("ansible", FormSpec{
		Positional: []PositionalSpec{{
			Label: "playbook or project path",
			// The connector reads both, and reads more from a project: a
			// single playbook gives plays, tasks and handlers, a directory
			// additionally gives roles, inventory and vault files.
			Desc:     "a playbook file, or a project directory with roles and inventory",
			Required: true,
		}},
	})

	registerSpec("bicep", FormSpec{
		Positional: []PositionalSpec{{
			Label:    "path",
			Desc:     "a .bicep file, a directory of them, or an ARM template JSON",
			Required: true,
		}},
	})

	registerSpec("cloudformation", FormSpec{
		Positional: []PositionalSpec{{
			Label:    "template path",
			Desc:     "the CloudFormation or AWS SAM template to scan",
			Required: true,
		}},
	})

	registerSpec("terraform", FormSpec{
		Positional: []PositionalSpec{
			{
				Label: "kind", Desc: "leave empty for HCL source",
				Options: []string{"plan", "state"},
			},
			{Label: "path", Desc: "directory or file to scan", Required: true},
		},
	})
}
