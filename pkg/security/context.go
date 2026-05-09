package security

import "context"

type jwtClaimsContextKey struct{}

func WithJWTClaimsContext(ctx context.Context, claims *JWTClaims) context.Context {
	if claims == nil {
		return ctx
	}
	return context.WithValue(ctx, jwtClaimsContextKey{}, *claims)
}

func JWTClaimsFromContext(ctx context.Context) (JWTClaims, bool) {
	if ctx == nil {
		return JWTClaims{}, false
	}
	v, ok := ctx.Value(jwtClaimsContextKey{}).(JWTClaims)
	return v, ok
}
