package routes

import (
	"context"
	"doubleboiler/models"
)

func can(ctx context.Context, target models.Organisation, role string) bool {
	if isAppAdmin(ctx) {
		return true
	}

	ou := orgUserFromContext(ctx, target)
	for _, r := range ou.Roles {
		if r.Can(role) {
			return true
		}
	}

	return false
}

func isAppAdmin(ctx context.Context) bool {
	if !isLoggedIn(ctx) {
		return false
	}
	return ctx.Value("user").(models.User).SuperAdmin
}
