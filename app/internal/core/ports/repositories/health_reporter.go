package repositories

import "context"

type HealthReporter interface {
	Health(ctx context.Context) error
}
