package view

import (
	"context"
	"soln-teachermodule/types"
)

func GetAuthenticatedUser(ctx context.Context) types.AuthenticatedUser {
	user, ok := ctx.Value(types.UserContextKey).(types.AuthenticatedUser)
	if !ok {
		return types.AuthenticatedUser{}
	}
	return user
}