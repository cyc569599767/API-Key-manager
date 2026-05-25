package storage

import (
	"os"
	"path/filepath"
)

type Paths struct {
	AppDir    string
	Database  string
	SecretKey string
}

func AppPaths() (Paths, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return Paths{}, err
	}
	appDir := filepath.Join(root, "KeyManager")
	if err := os.MkdirAll(appDir, 0o700); err != nil {
		return Paths{}, err
	}
	return Paths{
		AppDir:    appDir,
		Database:  filepath.Join(appDir, "keymanager.sqlite"),
		SecretKey: filepath.Join(appDir, "secret.key"),
	}, nil
}
