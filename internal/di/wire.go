//go:build wireinject
// +build wireinject

package di

import (
	"github.com/google/wire"

	"go-oauth-rbac-service/internal/app"
)

func InitializeApp() (*app.App, error) {
	panic(wire.Build(
		ConfigSet,
		ObservabilitySet,
		RuntimeInfraSet,
		RepositorySet,
		SecuritySet,
		ServiceSet,
		HTTPSet,
		AppSet,
	))
}

func InitializeMigrationRunner() (*MigrationRunner, error) {
	panic(wire.Build(
		ConfigSet,
		provideOpenDB,
		NewMigrationRunner,
	))
}
