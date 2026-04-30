package persistent

import (
	"github.com/tinboxw/skoll/internal/module/modgenerator"
	"github.com/tinboxw/skoll/internal/module/storageadapter/contracts"
)

// GeneratorRepository is a thin wrapper around modgenerator.Service. The
// service is fully stateless (template-driven artifact synthesis) so no SQL
// schema is required; the type exists for symmetry with the rest of the
// persistent adapter so all repositories can be obtained from a single
// factory.
type GeneratorRepository struct {
	inner *modgenerator.Service
}

// NewGeneratorRepository constructs a stateless generator repository.
func NewGeneratorRepository() *GeneratorRepository {
	return &GeneratorRepository{inner: modgenerator.NewService()}
}

// Models returns no models — generator is stateless.
func (r *GeneratorRepository) Models() []any { return nil }

// Generate delegates to the underlying generator service.
func (r *GeneratorRepository) Generate(module string) (modgenerator.Result, error) {
	return r.inner.Generate(module)
}

// GenerateWithSchema delegates to the underlying generator service.
func (r *GeneratorRepository) GenerateWithSchema(module string, schema *modgenerator.FormSchema, templateVersion string) (modgenerator.Result, error) {
	return r.inner.GenerateWithSchema(module, schema, templateVersion)
}

var _ contracts.GeneratorRepository = (*GeneratorRepository)(nil)
