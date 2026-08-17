package seed

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

type seedStep struct {
	name     string
	aliases  []string
	optional bool
	run      func(context.Context) error
}

func runSeedSteps(ctx context.Context, db *pgxpool.Pool, steps []seedStep) error {
	for _, step := range steps {
		applied, appliedName, err := isSeedStepApplied(ctx, db, step)
		if err != nil {
			slog.Warn("seed history check failed", "name", step.name, "error", err)
			if !step.optional {
				return fmt.Errorf("seed history check %s: %w", step.name, err)
			}
			continue
		}
		if applied {
			if appliedName != step.name {
				_ = markSeedApplied(ctx, db, step.name)
			}
			slog.Info("seed skipped (already applied)", "name", step.name, "applied_as", appliedName)
			continue
		}

		slog.Info("seed applying", "name", step.name)
		if err := step.run(ctx); err != nil {
			slog.Warn("seed failed", "name", step.name, "error", err)
			if !step.optional {
				return fmt.Errorf("seed %s: %w", step.name, err)
			}
			continue
		}
		if err := markSeedApplied(ctx, db, step.name); err != nil {
			slog.Warn("failed to mark seed", "name", step.name, "error", err)
			if !step.optional {
				return fmt.Errorf("mark seed %s: %w", step.name, err)
			}
			continue
		}
		slog.Info("seed applied", "name", step.name)
	}
	return nil
}

func isSeedStepApplied(ctx context.Context, db *pgxpool.Pool, step seedStep) (bool, string, error) {
	names := append([]string{step.name}, step.aliases...)
	for _, name := range names {
		applied, err := isSeedApplied(ctx, db, name)
		if err != nil {
			return false, "", err
		}
		if applied {
			return true, name, nil
		}
	}
	return false, "", nil
}
