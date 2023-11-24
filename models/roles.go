package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

var ValidRoles = map[string]Role{
	"admin":    adminRole,
	"teamlead": teamleadRole,
}

var adminRole = Role{
	Name:    "admin",
	Label:   "Admin",
	implies: Roles{teamleadRole},
}

var teamleadRole = Role{
	Name:    "teamlead",
	Label:   "Team Lead",
	implies: Roles{},
}

type Roles []Role

func (roles Roles) Value() (driver.Value, error) {
	return json.Marshal(roles)
}

func (roles *Roles) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &roles)
}

func (roles Roles) Can(name string) bool {
	for _, role := range roles {
		if role.Can(name) {
			return true
		}
	}
	return false
}

type Role struct {
	Name    string
	Label   string
	implies Roles
}

func (this *Role) Can(role string) bool {
	if role == this.Name {
		return true
	}
	this.implies = ValidRoles[this.Name].implies
	for _, sub := range this.implies {
		if sub.Can(role) {
			return true
		}
	}
	return false
}

func (this Role) Implications() []string {
	ret := []string{}
	for name, _ := range ValidRoles {
		if name == this.Name {
			continue
		}
		if this.Can(name) {
			ret = append(ret, name)
		}
	}
	return ret
}
