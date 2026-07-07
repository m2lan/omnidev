package service

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omnidev/go-common/logger"

	"github.com/omnidev/gateway/internal/adapter"
	"github.com/omnidev/gateway/internal/repository"
)

// AdapterResolver resolves an AI adapter for a given model, checking user configs first
// then falling back to the global adapter registry.
//
// Resolution strategy (3-pass):
//  1. Exact model match in user configs
//  2. Default config with empty models (supports all models)
//  3. Any config with empty models
//  4. Fallback to global registry
type AdapterResolver struct {
	userConfigRepo repository.UserAIConfigRepository
	adapterFactory *adapter.Factory
	adapters       *adapter.Registry
}

// NewAdapterResolver creates a new adapter resolver.
func NewAdapterResolver(
	userConfigRepo repository.UserAIConfigRepository,
	adapterFactory *adapter.Factory,
	adapters *adapter.Registry,
) *AdapterResolver {
	return &AdapterResolver{
		userConfigRepo: userConfigRepo,
		adapterFactory: adapterFactory,
		adapters:       adapters,
	}
}

// Resolve resolves an adapter for the given user and model.
// User configs are checked first (higher priority), then global registry.
func (r *AdapterResolver) Resolve(ctx context.Context, userID uuid.UUID, modelID string) (adapter.Adapter, error) {
	if r.userConfigRepo != nil && r.adapterFactory != nil {
		userConfigs, err := r.userConfigRepo.ListByUserID(ctx, userID)
		if err == nil {
			// First pass: look for exact model match
			for _, cfg := range userConfigs {
				for _, m := range cfg.Models {
					if m == modelID {
						adp, err := r.adapterFactory.CreateAdapter(cfg)
						if err != nil {
							logger.Log.Warn("Failed to create adapter from user config",
								zap.String("provider", cfg.Provider),
								zap.Error(err),
							)
							continue
						}
						return adp, nil
					}
				}
			}

			// Second pass: use default config with empty models (supports all models)
			for _, cfg := range userConfigs {
				if cfg.IsDefault && len(cfg.Models) == 0 {
					adp, err := r.adapterFactory.CreateAdapter(cfg)
					if err != nil {
						logger.Log.Warn("Failed to create adapter from default config",
							zap.String("provider", cfg.Provider),
							zap.Error(err),
						)
						break
					}
					return adp, nil
				}
			}

			// Third pass: use any config with empty models
			for _, cfg := range userConfigs {
				if len(cfg.Models) == 0 {
					adp, err := r.adapterFactory.CreateAdapter(cfg)
					if err != nil {
						logger.Log.Warn("Failed to create adapter from config",
							zap.String("provider", cfg.Provider),
							zap.Error(err),
						)
						continue
					}
					return adp, nil
				}
			}
		}
	}

	// Fallback to global registry
	return r.adapters.GetForModel(modelID)
}
