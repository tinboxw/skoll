package hostservice

import (
	"context"
	"strings"

	"github.com/tinboxw/skoll/pkg/security"
)

type hostActor struct {
	typ  string
	id   string
	name string
}

func trustedHostActor(ctx context.Context, pluginID string) hostActor {
	if claims, ok := security.JWTClaimsFromContext(ctx); ok && strings.TrimSpace(claims.Subject) != "" {
		return hostActor{typ: "user", id: strings.TrimSpace(claims.Subject)}
	}
	return hostActor{typ: "plugin", id: "plugin:" + pluginID, name: pluginID}
}
