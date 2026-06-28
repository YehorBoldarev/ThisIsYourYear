package mongo

import (
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"

	"kibou/thisisyouryear/internal/domain"
)

// translate converts MongoDB driver errors into domain errors so that storage
// specifics never leak past this package. Unrecognized errors are wrapped with
// context and returned as-is.
func translate(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, mongo.ErrNoDocuments):
		return domain.ErrNotFound
	case mongo.IsDuplicateKeyError(err):
		return fmt.Errorf("duplicate key: %w", domain.ErrConflict)
	default:
		return fmt.Errorf("mongo: %w", err)
	}
}
