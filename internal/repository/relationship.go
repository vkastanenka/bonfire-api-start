package repository

// type Relationship struct {
// 	q db.Querier
// }

// func NewRelationship(q db.Querier) *Relationship {
// 	return &Relationship{q: q}
// }

// func (r *Relationship) ListByUserID(ctx context.Context, userID uuid.UUID, relType relationship.Type) ([]relationship.Relationship, error) {
// 	if userID == uuid.Nil {
// 		return []relationship.Relationship{}, nil
// 	}

// 	dbUUID := pgtype.UUID{Bytes: userID, Valid: true}
// 	var rows []db.Relationship
// 	var err error

// 	switch relType {
// 	case relationship.TypeFriends:
// 		rows, err = r.store.RelationshipsListFriendsByUserID(ctx, dbUUID)
// 	case relationship.TypeBlocked:
// 		rows, err = r.store.RelationshipsListBlockedByUserID(ctx, dbUUID)
// 	case relationship.TypePending:
// 		rows, err = r.store.RelationshipsListPendingByUserID(ctx, dbUUID)
// 	case relationship.TypeUnknown:
// 		rows, err = r.store.RelationshipsListByUserID(ctx, dbUUID)
// 	default:
// 		return nil, apperr.NewInvalidArgument(nil, apperr.WithMsg("invalid relationship type filter"))
// 	}

// 	if err != nil {
// 		if errors.Is(err, pgx.ErrNoRows) {
// 			return []relationship.Relationship{}, nil
// 		}
// 		return nil, apperr.NewInternal(err, apperr.WithMsg("failed to list relationships"))
// 	}

// 	relationships := make([]relationship.Relationship, len(rows))
// 	for i, row := range rows {
// 		relationships[i] = relationshipFromDB(row)
// 	}

// 	return relationships, nil
// }

// func (r *Relationship) Get(ctx context.Context, p relationship.GetParams) (relationship.Relationship, error) {
// 	row, err := r.store.RelationshipGet(ctx, db.RelationshipGetParams{
// 		User1ID: pgtype.UUID{Bytes: p.User1ID, Valid: true},
// 		User2ID: pgtype.UUID{Bytes: p.User2ID, Valid: true},
// 	})
// 	if err != nil {
// 		if errors.Is(err, pgx.ErrNoRows) {
// 			return relationship.Relationship{}, apperr.NewNotFound(err, apperr.WithMsg("relationship not found"))
// 		}
// 		return relationship.Relationship{}, apperr.NewInternal(err, apperr.WithMsg("failed to fetch relationship"))
// 	}
// 	return relationshipFromDB(row), nil
// }

// func (r *Relationship) GetForUpdate(ctx context.Context, p relationship.GetParams) (relationship.Relationship, error) {
// 	row, err := r.store.RelationshipGetForUpdate(ctx, db.RelationshipGetForUpdateParams{
// 		User1ID: pgtype.UUID{Bytes: p.User1ID, Valid: true},
// 		User2ID: pgtype.UUID{Bytes: p.User2ID, Valid: true},
// 	})
// 	if err != nil {
// 		if errors.Is(err, pgx.ErrNoRows) {
// 			return relationship.Relationship{}, apperr.NewNotFound(err, apperr.WithMsg("relationship not found"))
// 		}
// 		return relationship.Relationship{}, apperr.NewInternal(err, apperr.WithMsg("failed to fetch relationship for update"))
// 	}
// 	return relationshipFromDB(row), nil
// }

// func (r *Relationship) Upsert(ctx context.Context, p relationship.UpsertParams) (relationship.Relationship, error) {
// 	row, err := r.store.RelationshipUpsert(ctx, db.RelationshipUpsertParams{
// 		User1ID: pgtype.UUID{Bytes: p.User1ID, Valid: true},
// 		User2ID: pgtype.UUID{Bytes: p.User2ID, Valid: true},
// 		Type:    int16(p.Type),
// 		ActorID: pgtype.UUID{Bytes: p.ActorID, Valid: true},
// 	})
// 	if err != nil {
// 		return relationship.Relationship{}, apperr.NewInternal(err, apperr.WithMsg("failed to upsert relationship"))
// 	}
// 	return relationshipFromDB(row), nil
// }

// func (r *Relationship) Delete(ctx context.Context, p relationship.DeleteParams) error {
// 	err := r.store.RelationshipDelete(ctx, db.RelationshipDeleteParams{
// 		User1ID: pgtype.UUID{Bytes: p.User1ID, Valid: true},
// 		User2ID: pgtype.UUID{Bytes: p.User2ID, Valid: true},
// 	})
// 	if err != nil {
// 		if errors.Is(err, pgx.ErrNoRows) {
// 			return apperr.NewNotFound(err, apperr.WithMsg("relationship not found"))
// 		}
// 		return apperr.NewInternal(err, apperr.WithMsg("failed to delete relationship"))
// 	}
// 	return nil
// }

// func (r *Relationship) DeleteVerified(ctx context.Context, p relationship.DeleteVerifiedParams) error {
// 	err := r.store.RelationshipDeleteVerified(ctx, db.RelationshipDeleteVerifiedParams{
// 		User1ID: pgtype.UUID{Bytes: p.User1ID, Valid: true},
// 		User2ID: pgtype.UUID{Bytes: p.User2ID, Valid: true},
// 		ActorID: pgtype.UUID{Bytes: p.ActorID, Valid: true},
// 	})
// 	if err != nil {
// 		if errors.Is(err, pgx.ErrNoRows) {
// 			return apperr.NewNotFound(err, apperr.WithMsg("relationship not found or actor unauthorized"))
// 		}
// 		return apperr.NewInternal(err, apperr.WithMsg("failed to delete relationship"))
// 	}
// 	return nil
// }

// func relationshipFromDB(row db.Relationship) relationship.Relationship {
// 	return relationship.Relationship{
// 		User1ID:   uuid.UUID(row.User1ID.Bytes),
// 		User2ID:   uuid.UUID(row.User2ID.Bytes),
// 		ActorID:   uuid.UUID(row.ActorID.Bytes),
// 		Type:      relationship.Type(row.Type),
// 		CreatedAt: row.CreatedAt.Time,
// 		UpdatedAt: row.UpdatedAt.Time,
// 	}
// }

// var _ relationship.Repository = (*Relationship)(nil)
