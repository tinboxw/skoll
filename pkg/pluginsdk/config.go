package pluginsdk

import "context"

type ConfigService interface {
	Get(ctx context.Context) (map[string]any, error)
	Replace(ctx context.Context, values map[string]any) (map[string]any, error)
}

type SecretService interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
}
