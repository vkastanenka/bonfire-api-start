package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"bonfire-api/internal/db"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
)

type userCache interface {
	Delete(ctx context.Context, userID fields.ID) error
	DeleteBatch(ctx context.Context, userIDs []fields.ID) error
	Get(ctx context.Context, userID fields.ID) (*user.User, error)
	GetBatch(ctx context.Context, userIDs []fields.ID) (map[fields.ID]*user.User, []fields.ID, error)
	Set(ctx context.Context, usr *user.User) error
	SetBatch(ctx context.Context, users []*user.User) error
}

type UserRepository struct {
	store *db.Store
	cache userCache
	sf    singleflight.Group
}

func NewUserRepository(store *db.Store, cache userCache) *UserRepository {
	return &UserRepository{
		store: store.WithEntity(db.EntityUser),
		cache: cache,
	}
}

func (r *UserRepository) Availability(ctx context.Context, email *user.Email, username *user.Username) (bool, bool, error) {
	var emailStr, usernameStr string
	if email != nil {
		emailStr = email.String()
	}
	if username != nil {
		usernameStr = username.String()
	}

	row, err := r.store.UserAvailability(ctx, db.UserAvailabilityParams{
		Email:    emailStr,
		Username: usernameStr,
	})
	if err != nil {
		return false, false, r.store.Err(err)
	}

	return row.EmailAvailable.Bool, row.UsernameAvailable.Bool, nil
}

func (r *UserRepository) Create(ctx context.Context, u *user.User) (*user.User, error) {
	row, err := r.store.UserCreate(ctx, db.UserCreateParams{
		ID:                     db.ToUUID(u.ID()),
		Email:                  u.Email().String(),
		Username:               u.Username().String(),
		DisplayName:            u.DisplayName().String(),
		PasswordHash:           u.PasswordHash().String(),
		Phone:                  db.ToTextPtr(u.Phone().StringPtr()),
		Bio:                    db.ToTextPtr(u.Bio().StringPtr()),
		AvatarUrl:              db.ToTextPtr(u.AvatarURL().StringPtr()),
		BannerColor:            db.ToTextPtr(u.BannerColor().StringPtr()),
		PreferredPresence:      db.ToInt2Ptr(u.PreferredPresence().Int16Ptr()),
		PreferredPresenceUntil: db.ToTimestamptzPtr(u.PreferredPresenceUntil().TimePtr()),
		VerifiedAt:             db.ToTimestamptzPtr(u.VerifiedAt().TimePtr()),
		DisabledAt:             db.ToTimestamptzPtr(u.DisabledAt().TimePtr()),
		DeleteScheduledAt:      db.ToTimestamptzPtr(u.DeleteScheduledAt().TimePtr()),
		CreatedAt:              db.ToTimestamptz(u.CreatedAt().Time()),
		UpdatedAt:              db.ToTimestamptz(u.UpdatedAt().Time()),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return userFromRow(row)
}

func (r *UserRepository) Get(ctx context.Context, id fields.ID) (*user.User, error) {
	row, err := r.store.UserGet(ctx, db.ToUUID(id.UUID()))
	if err != nil {
		return nil, r.store.Err(err)
	}

	return userFromRow(row)
}

func (r *UserRepository) GetBatch(ctx context.Context, ids []fields.ID, batchLimit int32) (map[fields.ID]*user.User, error) {
	if len(ids) == 0 {
		return make(map[fields.ID]*user.User), nil
	}

	uuids := make([]uuid.UUID, len(ids))
	for i, id := range ids {
		uuids[i] = id.UUID()
	}

	rows, err := r.store.UserGetBatch(ctx, db.UserGetBatchParams{
		Ids:        db.ToUUIDs(uuids),
		BatchLimit: batchLimit,
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	result := make(map[fields.ID]*user.User, len(rows))
	for _, row := range rows {
		u, err := userFromRow(row)
		if err != nil {
			return nil, err
		}
		result[u.ID()] = u
	}

	return result, nil
}

func (r *UserRepository) GetCached(ctx context.Context, id fields.ID) (*user.User, error) {
	u, err := r.cache.Get(ctx, id)
	if err == nil && u != nil {
		return u, nil
	}

	if err != nil && !errors.Is(err, redis.ErrCacheMiss) {
		slog.WarnContext(ctx, "user cache read failed, falling back to database",
			"user_id", id.String(),
			"error", err,
			"scope", redis.ScopeUser,
		)
	}

	key := id.String()

	sfCtx := context.WithoutCancel(ctx)

	val, err, _ := r.sf.Do(key, func() (any, error) {
		dbUser, repoErr := r.Get(sfCtx, id)
		if repoErr != nil {
			return nil, repoErr
		}

		if cacheErr := r.cache.Set(sfCtx, dbUser); cacheErr != nil {
			slog.WarnContext(sfCtx, "failed to backfill user cache",
				"user_id", dbUser.ID().String(),
				"error", cacheErr,
				"scope", redis.ScopeUser,
			)
		}

		return dbUser, nil
	})

	if err != nil {
		return nil, err
	}

	return val.(*user.User), nil
}

func (r *UserRepository) GetCachedBatch(ctx context.Context, ids []fields.ID, batchLimit int32) (map[fields.ID]*user.User, error) {
	if len(ids) == 0 {
		return make(map[fields.ID]*user.User), nil
	}

	cachedUsers, missingIDs, err := r.cache.GetBatch(ctx, ids)
	if err != nil {
		slog.WarnContext(ctx, "user cache batch read failed, falling back to database",
			"requested_count", len(ids),
			"error", err,
			"scope", redis.ScopeUser,
		)
		missingIDs = ids
		cachedUsers = make(map[fields.ID]*user.User)
	}

	if len(missingIDs) == 0 {
		return cachedUsers, nil
	}

	sfKey := fmt.Sprintf("get_cached_batch:%d:%v", len(missingIDs), missingIDs)
	sfCtx := context.WithoutCancel(ctx)

	val, err, _ := r.sf.Do(sfKey, func() (any, error) {
		dbUsers, repoErr := r.GetBatch(sfCtx, missingIDs, batchLimit)
		if repoErr != nil {
			return nil, repoErr
		}

		if len(dbUsers) > 0 {
			toBackfill := make([]*user.User, 0, len(dbUsers))
			for _, u := range dbUsers {
				toBackfill = append(toBackfill, u)
			}

			if cacheErr := r.cache.SetBatch(sfCtx, toBackfill); cacheErr != nil {
				slog.WarnContext(sfCtx, "failed to backfill user batch cache",
					"count", len(toBackfill),
					"error", cacheErr,
					"scope", redis.ScopeUser,
				)
			}
		}

		return dbUsers, nil
	})

	if err != nil {
		if len(cachedUsers) > 0 {
			return cachedUsers, nil
		}
		return nil, err
	}

	fetchedUsers := val.(map[fields.ID]*user.User)
	for id, u := range fetchedUsers {
		cachedUsers[id] = u
	}

	return cachedUsers, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email user.Email) (*user.User, error) {
	row, err := r.store.UserGetByEmail(ctx, email.String())
	if err != nil {
		return nil, r.store.Err(err)
	}

	return userFromRow(row)
}

func (r *UserRepository) GetDeleteScheduledBatch(ctx context.Context, currentTime fields.Timestamp, batchLimit int32) ([]*user.User, error) {
	rows, err := r.store.UserGetDeleteScheduledBatch(ctx, db.UserGetDeleteScheduledBatchParams{
		Now:        db.ToTimestamptz(currentTime.Time()),
		BatchLimit: batchLimit,
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	users := make([]*user.User, 0, len(rows))
	for _, row := range rows {
		u, err := userFromRow(row)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	return users, nil
}

func (r *UserRepository) Update(ctx context.Context, u *user.User) (*user.User, error) {
	row, err := r.store.UserUpdate(ctx, db.UserUpdateParams{
		ID:                     db.ToUUID(u.ID()),
		Email:                  u.Email().String(),
		Username:               u.Username().String(),
		DisplayName:            u.DisplayName().String(),
		PasswordHash:           u.PasswordHash().String(),
		Phone:                  db.ToTextPtr(u.Phone().StringPtr()),
		Bio:                    db.ToTextPtr(u.Bio().StringPtr()),
		AvatarUrl:              db.ToTextPtr(u.AvatarURL().StringPtr()),
		BannerColor:            db.ToTextPtr(u.BannerColor().StringPtr()),
		PreferredPresence:      db.ToInt2Ptr(u.PreferredPresence().Int16Ptr()),
		PreferredPresenceUntil: db.ToTimestamptzPtr(u.PreferredPresenceUntil().TimePtr()),
		VerifiedAt:             db.ToTimestamptzPtr(u.VerifiedAt().TimePtr()),
		DisabledAt:             db.ToTimestamptzPtr(u.DisabledAt().TimePtr()),
		DeleteScheduledAt:      db.ToTimestamptzPtr(u.DeleteScheduledAt().TimePtr()),
		UpdatedAt:              db.ToTimestamptz(u.UpdatedAt().Time()),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return userFromRow(row)
}

func (r *UserRepository) UpdateBatch(ctx context.Context, usersJson []byte) ([]*user.User, error) {
	rows, err := r.store.UserUpdateBatch(ctx, usersJson)
	if err != nil {
		return nil, r.store.Err(err)
	}

	updatedUsers := make([]*user.User, 0, len(rows))
	for _, row := range rows {
		u, err := userFromRow(row)
		if err != nil {
			return nil, err
		}
		updatedUsers = append(updatedUsers, u)
	}

	return updatedUsers, nil
}

func userFromRow(row db.User) (*user.User, error) {
	userID := db.FromUUID[uuid.UUID](row.ID)
	userIDStr := userID.String()

	mapErr := func(msg, key string, val any, err error) *errs.Error {
		return errs.Internal(msg).
			Wrap(err).
			Reason("CORRUPT_DATABASE_RECORD").
			Meta(key, fmt.Sprintf("%v", val)).
			Resource("User", userIDStr, "", "database row mapping")
	}

	id, err := fields.ParseRequiredID("id", userID)
	if err != nil {
		return nil, mapErr("failed to parse user id from database", "id", userIDStr, err)
	}

	email, err := user.ParseEmail("email", row.Email)
	if err != nil {
		return nil, mapErr("failed to parse user email from database", "email", row.Email, err)
	}

	username, err := user.ParseUsername("username", row.Username)
	if err != nil {
		return nil, mapErr("failed to parse username from database", "username", row.Username, err)
	}

	passwordHash, err := user.ParsePasswordHash("password_hash", row.PasswordHash)
	if err != nil {
		return nil, mapErr("failed to parse password hash from database", "password_hash", row.PasswordHash, err)
	}

	displayName, err := user.ParseDisplayName("display_name", row.DisplayName)
	if err != nil {
		return nil, mapErr("failed to parse display name from database", "display_name", row.DisplayName, err)
	}

	phone, err := user.ParsePhone("phone", db.FromText[string](row.Phone))
	if err != nil {
		return nil, mapErr("failed to parse phone from database", "phone", row.Phone.String, err)
	}

	bio, err := user.ParseBio("bio", db.FromText[string](row.Bio))
	if err != nil {
		return nil, mapErr("failed to parse bio from database", "bio", row.Bio.String, err)
	}

	avatarURL, err := fields.ParseURL("avatar_url", db.FromText[string](row.AvatarUrl))
	if err != nil {
		return nil, mapErr("failed to parse avatar url from database", "avatar_url", row.AvatarUrl.String, err)
	}

	bannerColor, err := fields.ParseHexColor("banner_color", db.FromText[string](row.BannerColor))
	if err != nil {
		return nil, mapErr("failed to parse banner color from database", "banner_color", row.BannerColor.String, err)
	}

	rowPreferredPresence := db.FromInt2[int16](row.PreferredPresence)
	preferredPresence, err := user.ParsePreferredPresenceFromInt16("preferred_presence", rowPreferredPresence)
	if err != nil {
		return nil, mapErr("failed to parse preferred presence from database", "preferred_presence", rowPreferredPresence, err)
	}

	preferredPresenceUntil := fields.NewTimestampFromTime(db.FromTimestamptz(row.PreferredPresenceUntil))
	verifiedAt := fields.NewTimestampFromTime(db.FromTimestamptz(row.VerifiedAt))
	disabledAt := fields.NewTimestampFromTime(db.FromTimestamptz(row.DisabledAt))
	deleteScheduledAt := fields.NewTimestampFromTime(db.FromTimestamptz(row.DeleteScheduledAt))
	createdAt := fields.NewTimestampFromTime(db.FromTimestamptz(row.CreatedAt))
	updatedAt := fields.NewTimestampFromTime(db.FromTimestamptz(row.UpdatedAt))

	return user.Reconstitute(
		id,
		email,
		username,
		passwordHash,
		phone,
		displayName,
		bio,
		avatarURL,
		bannerColor,
		preferredPresence,
		preferredPresenceUntil,
		verifiedAt,
		disabledAt,
		deleteScheduledAt,
		createdAt,
		updatedAt,
	), nil
}
