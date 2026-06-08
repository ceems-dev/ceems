package lsf

import (
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/ceems-dev/ceems/pkg/api/base"
	"github.com/ceems-dev/ceems/pkg/api/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	noOpLogger = slog.New(slog.DiscardHandler)
	start, _   = time.Parse(base.DatetimezoneLayout, "2026-04-12T18:15:00+0100")
	end, _     = time.Parse(base.DatetimezoneLayout, "2026-04-12T18:30:00+0100")
	current, _ = time.Parse(base.DatetimezoneLayout, "2026-04-12T18:30:00+0100")
)

func TestLSFFetcherMultiCluster(t *testing.T) {
	// Add bacct command to PATH
	bacctPath, _ := filepath.Abs("../../testdata")

	// mock config
	clusters := []models.Cluster{
		{
			ID:      "lsf-0",
			Manager: "lsf",
			CLI:     models.CLIConfig{Path: bacctPath},
		},
		{
			ID:      "lsf-1",
			Manager: "lsf",
			CLI:     models.CLIConfig{Path: bacctPath},
		},
	}

	ctx := t.Context()

	for _, cluster := range clusters {
		lsf, err := New(cluster, noOpLogger)
		require.NoError(t, err)

		clusterJobs, err := lsf.FetchUnits(ctx, start, end)
		require.NoError(t, err)
		assert.Len(t, clusterJobs[0].Units, 8, "Num jobs")

		clusterUsers, clusterProjects, err := lsf.FetchUsersProjects(ctx, current)
		require.NoError(t, err)
		assert.Len(t, clusterUsers[0].Users, 5, "Num Users")
		assert.Len(t, clusterProjects[0].Projects, 4, "Num Projects")
	}
}

func TestFetchUserProjects(t *testing.T) {
	// Cluster ID
	clusterID := "lsf-0"

	// Make a manager struct
	manager := lsfScheduler{
		logger: noOpLogger,
		cluster: models.Cluster{
			ID:      clusterID,
			Manager: lsfBatchScheduler,
		},
		lastFetchedJobs: []models.Unit{
			{
				User:    "usr1",
				Project: "prj1",
			},
			{
				User:    "usr1",
				Project: "prj1",
			},
			{
				User:    "usr2",
				Project: "prj1",
			},
			{
				User:    "usr2",
				Project: "prj2",
			},
			{
				User:    "usr3",
				Project: "prj2",
			},
		},
	}

	// Expected users
	expectedUsers := []models.User{
		{
			ClusterID:       clusterID,
			ResourceManager: lsfBatchScheduler,
			Name:            "usr1",
			Projects:        models.List{"prj1"},
			LastUpdatedAt:   "2026-04-12T18:30:00+0100",
		},
		{
			ClusterID:       clusterID,
			ResourceManager: lsfBatchScheduler,
			Name:            "usr2",
			Projects:        models.List{"prj1", "prj2"},
			LastUpdatedAt:   "2026-04-12T18:30:00+0100",
		},
		{
			ClusterID:       clusterID,
			ResourceManager: lsfBatchScheduler,
			Name:            "usr3",
			Projects:        models.List{"prj2"},
			LastUpdatedAt:   "2026-04-12T18:30:00+0100",
		},
	}
	expectedProjects := []models.Project{
		{
			ClusterID:       clusterID,
			ResourceManager: lsfBatchScheduler,
			Name:            "prj1",
			Users:           models.List{"usr1", "usr2"},
			LastUpdatedAt:   "2026-04-12T18:30:00+0100",
		},
		{
			ClusterID:       clusterID,
			ResourceManager: lsfBatchScheduler,
			Name:            "prj2",
			Users:           models.List{"usr2", "usr3"},
			LastUpdatedAt:   "2026-04-12T18:30:00+0100",
		},
	}

	// Get user and project models
	users, projects := manager.fetchUserProjects(current)

	assert.ElementsMatch(t, expectedUsers, users)
	assert.ElementsMatch(t, expectedProjects, projects)
}
