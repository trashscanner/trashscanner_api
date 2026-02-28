// Package rbac provides role-based access control middleware.
package rbac

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/trashscanner/trashscanner_api/internal/errlocal"
	"github.com/trashscanner/trashscanner_api/internal/models"
	"github.com/trashscanner/trashscanner_api/internal/utils"
)

// RequireRole returns middleware that allows only users with specific roles.
func RequireRole(writeError func(http.ResponseWriter, *http.Request, error), roles ...models.Role) mux.MiddlewareFunc {
	allowed := make(map[models.Role]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := utils.GetUser(r.Context())
			if _, ok := allowed[user.Role]; !ok {
				writeError(w, r, errlocal.NewErrForbidden("access denied", "insufficient role", nil))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
