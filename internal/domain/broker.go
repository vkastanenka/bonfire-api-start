package domain

import "github.com/jackc/pgx/v5/pgtype"

type MessageBroker interface {
	PublishUserRegisteredEvent(userID pgtype.UUID, email string)
}
