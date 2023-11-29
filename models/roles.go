package models

import (
	scumrole "github.com/davidbanham/scum/role"
)

type Role = scumrole.Role
type Roles = scumrole.Roles

var ValidRoles = Roles{
	adminRole,
	teamleadRole,
}

var adminRole = Role{
	Name:    "admin",
	Label:   "Admin",
	Implies: Roles{teamleadRole},
}

var teamleadRole = Role{
	Name:    "teamlead",
	Label:   "Team Lead",
	Implies: Roles{},
}
