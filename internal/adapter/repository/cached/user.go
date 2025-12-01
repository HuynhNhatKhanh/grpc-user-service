package cached

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	"grpc-user-service/internal/adapter/cache"
	domain "grpc-user-service/internal/domain/user"
	"grpc-user-service/internal/usecase/user"
)

// CachedUserRepository implements user.Repository with caching support.
// It wraps a persistent repository (DB) and a cache implementation.
type CachedUserRepository struct {
	dbRepo user.Repository
	cache  cache.UserCache
	log    *zap.Logger
	group  singleflight.Group
}

// NewCachedUserRepository creates a new instance of CachedUserRepository.
func NewCachedUserRepository(dbRepo user.Repository, cache cache.UserCache, log *zap.Logger) user.Repository {
	return &CachedUserRepository{
		dbRepo: dbRepo,
		cache:  cache,
		log:    log,
	}
}

// Create delegates to the DB repository.
func (r *CachedUserRepository) Create(ctx context.Context, u *domain.User) (int64, error) {
	return r.dbRepo.Create(ctx, u)
}

// GetByID retrieves a user by ID using Cache-Aside pattern.
func (r *CachedUserRepository) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	// Try to get from cache first
	if r.cache != nil {
		cachedUser, err := r.cache.Get(ctx, id)
		if err != nil {
			r.log.Warn("cache get error, falling back to database", zap.Int64("id", id), zap.Error(err))
		} else if cachedUser != nil {
			r.log.Debug("user retrieved from cache", zap.Int64("id", id))
			return cachedUser, nil
		}
	}

	// Cache miss or cache disabled - use single-flight to prevent stampede
	key := fmt.Sprintf("user:%d", id)
	result, err, _ := r.group.Do(key, func() (any, error) {
		// Double-check cache in case another request populated it while we were waiting
		if r.cache != nil {
			cachedUser, err := r.cache.Get(ctx, id)
			if err == nil && cachedUser != nil {
				r.log.Debug("user retrieved from cache after single-flight wait", zap.Int64("id", id))
				return cachedUser, nil
			}
		}

		// Only one request hits database
		u, err := r.dbRepo.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}

		// Store in cache for future requests
		if r.cache != nil {
			if err := r.cache.Set(ctx, u); err != nil {
				r.log.Warn("failed to cache user", zap.Int64("id", id), zap.Error(err))
			}
		}

		return u, nil
	})

	if err != nil {
		return nil, err
	}

	return result.(*domain.User), nil
}

// GetByEmail delegates to the DB repository.
func (r *CachedUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.dbRepo.GetByEmail(ctx, email)
}

// Update updates the user in DB and invalidates the cache.
func (r *CachedUserRepository) Update(ctx context.Context, u *domain.User) (int64, error) {
	id, err := r.dbRepo.Update(ctx, u)
	if err != nil {
		return 0, err
	}

	// Invalidate cache after successful update
	if r.cache != nil {
		if err := r.cache.Delete(ctx, u.ID); err != nil {
			r.log.Warn("failed to invalidate cache after update", zap.Int64("id", u.ID), zap.Error(err))
		}
	}

	return id, nil
}

// Delete deletes the user from DB and invalidates the cache.
func (r *CachedUserRepository) Delete(ctx context.Context, id int64) (int64, error) {
	deletedID, err := r.dbRepo.Delete(ctx, id)
	if err != nil {
		return 0, err
	}

	// Invalidate cache after successful deletion
	if r.cache != nil {
		if err := r.cache.Delete(ctx, id); err != nil {
			r.log.Warn("failed to invalidate cache after delete", zap.Int64("id", id), zap.Error(err))
		}
	}

	return deletedID, nil
}

// List delegates to the DB repository.
func (r *CachedUserRepository) List(ctx context.Context, query string, page, limit int64) ([]domain.User, int64, error) {
	return r.dbRepo.List(ctx, query, page, limit)
}
