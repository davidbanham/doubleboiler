package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuditsByOrg(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	org := organisationFixture()
	org.Save(ctx)

	fix := userFixture()
	assert.Nil(t, fix.Save(ctx))

	audits := Audits{}
	assert.Nil(t, audits.FindAll(ctx, Criteria{Query: &ByOrg{ID: org.ID}}))

	closeTx(t, ctx)
}

func TestAuditsByEntity(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	org := organisationFixture()
	org.Save(ctx)

	fix := userFixture()
	assert.Nil(t, fix.Save(ctx))

	audits := Audits{}
	criteria := Criteria{}
	AddCustomQuery(ByEntityID{EntityID: fix.ID}, &criteria)
	assert.Nil(t, audits.FindAll(ctx, criteria))

	closeTx(t, ctx)
}
