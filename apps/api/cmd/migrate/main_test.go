package main

import (
	"errors"
	"io/fs"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/stretchr/testify/require"
)

func TestCommand(t *testing.T) {
	t.Parallel()

	for _, command := range []string{"up", "down", "version"} {
		t.Run(command, func(t *testing.T) {
			t.Parallel()

			got, err := parseCommand([]string{command})
			require.NoError(t, err)
			require.Equal(t, command, got)
		})
	}

	for _, tt := range []struct {
		name string
		args []string
	}{
		{name: "missing", args: nil},
		{name: "invalid", args: []string{"invalid"}},
		{name: "extra", args: []string{"up", "extra"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := parseCommand(tt.args)
			require.EqualError(t, err, "command must be one of: up, down, version")
		})
	}
}

func TestIsEmptyMigrationNoOp(t *testing.T) {
	firstError := &fs.PathError{Op: "first", Path: "migrations", Err: fs.ErrNotExist}
	require.True(t, isEmptyMigrationNoOp(migrate.ErrNilVersion, firstError))
	require.False(t, isEmptyMigrationNoOp(migrate.ErrNilVersion, &fs.PathError{Op: "read", Path: "migrations", Err: fs.ErrNotExist}))
	require.False(t, isEmptyMigrationNoOp(nil, firstError))
	require.False(t, isEmptyMigrationNoOp(migrate.ErrNilVersion, errors.New("source is unavailable")))
}
