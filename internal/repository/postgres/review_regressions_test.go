package postgres_test

import (
	"context"
	"os"
	"testing"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/repository/postgres"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// Opt-in only: use a disposable database with the main migrations applied.
func TestReviewOrganizationNotificationRecipients(t *testing.T) {
	dsn := os.Getenv("REVIEW_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("requires an isolated review database")
	}
	ctx := context.Background()
	p, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	defer p.Close()
	var database string
	require.NoError(t, p.QueryRow(ctx, "SELECT current_database()").Scan(&database))
	require.Equal(t, "review", database, "refusing to use a non-review database")

	orgRepo := postgres.NewValuationOrgRepo(p)
	org := &domain.ValuationOrg{Name: uuid.NewString(), IsActive: true}
	require.NoError(t, orgRepo.Create(ctx, org, []domain.ValuationOrgEmployee{
		{Username: uuid.NewString(), Password: "stored-hash", FirstName: "Recipient"},
		{FirstName: "No login"},
	}))
	defer func() { _, _ = p.Exec(ctx, "DELETE FROM valuation_org_profile WHERE id=$1", org.ID) }()
	var employeeID domain.ID
	require.NoError(t, p.QueryRow(ctx, "SELECT id FROM valuation_org_employee WHERE org_id=$1 AND username IS NOT NULL", org.ID).Scan(&employeeID))
	land := postgres.NewLandAcquisitionRepo(p)
	ids, err := land.OrgUserIDs(ctx, org.ID)
	require.NoError(t, err)
	require.Equal(t, []domain.ID{employeeID}, ids)

	_, err = p.Exec(ctx, "UPDATE valuation_org_employee SET is_active=false WHERE id=$1", employeeID)
	require.NoError(t, err)
	ids, err = land.OrgUserIDs(ctx, org.ID)
	require.NoError(t, err)
	require.Empty(t, ids)
}
