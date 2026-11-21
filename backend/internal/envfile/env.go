// Package envfile loads the repository's supplied configuration without replacing process overrides.
package envfile

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

var ErrRootNotFound = errors.New("repository root not found")

func Root() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "backend", "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrRootNotFound
		}
		dir = parent
	}
}

func Load() error {
	root, err := Root()
	if errors.Is(err, ErrRootNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	err = godotenv.Load(filepath.Join(root, ".env"))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
