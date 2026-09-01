package update

import (
	"context"
	"fmt"

	"github.com/creativeprojects/go-selfupdate"

	"github.com/vesvai/vesvai/internal/core/config"
)

const repoSlug = "vesvai/vesvai"

type Release struct {
	Version string
}

func DetectLatest(ctx context.Context) (*Release, bool, error) {
	latest, found, err := selfupdate.DetectLatest(ctx, selfupdate.ParseSlug(repoSlug))
	if err != nil {
		return nil, false, fmt.Errorf("detect latest: %w", err)
	}
	if !found {
		return nil, false, nil
	}
	return &Release{
		Version: latest.Version(),
	}, true, nil
}

func IsNewerThan(current, latest string) bool {
	latestRelease, found, err := selfupdate.DetectVersion(context.Background(), selfupdate.ParseSlug(repoSlug), latest)
	if err != nil || !found {
		return false
	}
	return latestRelease.GreaterThan(current)
}

func UpdateToLatest(ctx context.Context) error {
	latest, found, err := selfupdate.DetectLatest(ctx, selfupdate.ParseSlug(repoSlug))
	if err != nil {
		return fmt.Errorf("detect latest: %w", err)
	}
	if !found {
		return fmt.Errorf("no release found")
	}

	exe, err := selfupdate.ExecutablePath()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}

	if err := selfupdate.UpdateTo(ctx, latest.AssetURL, latest.AssetName, exe); err != nil {
		return fmt.Errorf("update binary: %w", err)
	}

	return nil
}

func CurrentVersion() string {
	return config.AppVersion
}
