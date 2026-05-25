package backup

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"keymanager/internal/storage"

	_ "modernc.org/sqlite"
)

const (
	databaseName     = "keymanager.sqlite"
	secretKeyName    = "secret.key"
	manifestName     = "manifest.json"
	maxDatabaseBytes = 512 * 1024 * 1024
	maxSecretBytes   = 4096
	maxManifestBytes = 64 * 1024
)

type ValidatedBackup struct {
	Database  []byte
	SecretKey []byte
}

type manifest struct {
	App       string   `json:"app"`
	Version   int      `json:"version"`
	CreatedAt string   `json:"createdAt"`
	Files     []string `json:"files"`
}

func CreateBackup(zipPath string, paths storage.Paths) error {
	database, err := os.ReadFile(paths.Database)
	if err != nil {
		return fmt.Errorf("读取数据库文件失败: %w", err)
	}
	secretKey, err := os.ReadFile(paths.SecretKey)
	if err != nil {
		return fmt.Errorf("读取密钥文件失败: %w", err)
	}

	file, err := os.OpenFile(zipPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := zip.NewWriter(file)
	if err := writeEntry(writer, databaseName, database); err != nil {
		writer.Close()
		return err
	}
	if err := writeEntry(writer, secretKeyName, secretKey); err != nil {
		writer.Close()
		return err
	}
	metadata, err := json.MarshalIndent(manifest{
		App:       "KeyManager",
		Version:   1,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Files:     []string{databaseName, secretKeyName},
	}, "", "  ")
	if err != nil {
		writer.Close()
		return err
	}
	if err := writeEntry(writer, manifestName, metadata); err != nil {
		writer.Close()
		return err
	}
	return writer.Close()
}

func ValidateBackup(zipPath string) (*ValidatedBackup, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, errors.New("备份文件不是有效的 zip 文件")
	}
	defer reader.Close()

	seen := map[string]bool{}
	validated := &ValidatedBackup{}
	for _, file := range reader.File {
		if err := validateEntryName(file.Name, file.FileInfo().IsDir()); err != nil {
			return nil, err
		}
		if seen[file.Name] {
			return nil, fmt.Errorf("备份文件无效：重复的 %s", file.Name)
		}
		seen[file.Name] = true

		limit, ok := sizeLimit(file.Name)
		if !ok {
			return nil, fmt.Errorf("备份文件无效：包含不支持的文件 %s", file.Name)
		}
		content, err := readZipFile(file, limit)
		if err != nil {
			return nil, err
		}
		switch file.Name {
		case databaseName:
			validated.Database = content
		case secretKeyName:
			validated.SecretKey = content
		}
	}
	if len(validated.Database) == 0 {
		return nil, errors.New("备份文件无效：缺少 keymanager.sqlite")
	}
	if len(validated.SecretKey) == 0 {
		return nil, errors.New("备份文件无效：缺少 secret.key")
	}
	if err := validateSecretKey(validated.SecretKey); err != nil {
		return nil, err
	}
	if err := validateDatabase(validated.Database); err != nil {
		return nil, err
	}
	return validated, nil
}

func RestoreBackup(validated *ValidatedBackup, paths storage.Paths) error {
	if validated == nil {
		return errors.New("备份内容为空")
	}
	if err := os.MkdirAll(paths.AppDir, 0o700); err != nil {
		return err
	}
	databaseBackup := paths.Database + ".restore-bak"
	secretBackup := paths.SecretKey + ".restore-bak"
	_ = os.Remove(databaseBackup)
	_ = os.Remove(secretBackup)

	databaseMoved, err := moveIfExists(paths.Database, databaseBackup)
	if err != nil {
		return err
	}
	secretMoved, err := moveIfExists(paths.SecretKey, secretBackup)
	if err != nil {
		rollback(paths.Database, databaseBackup, databaseMoved, paths.SecretKey, secretBackup, false)
		return err
	}

	if err := os.WriteFile(paths.Database, validated.Database, 0o600); err != nil {
		rollback(paths.Database, databaseBackup, databaseMoved, paths.SecretKey, secretBackup, secretMoved)
		return err
	}
	if err := os.WriteFile(paths.SecretKey, validated.SecretKey, 0o600); err != nil {
		rollback(paths.Database, databaseBackup, databaseMoved, paths.SecretKey, secretBackup, secretMoved)
		return err
	}
	return nil
}

func CommitRestore(paths storage.Paths) {
	_ = os.Remove(paths.Database + ".restore-bak")
	_ = os.Remove(paths.SecretKey + ".restore-bak")
}

func RollbackRestore(paths storage.Paths) {
	rollback(paths.Database, paths.Database+".restore-bak", true, paths.SecretKey, paths.SecretKey+".restore-bak", true)
}

func writeEntry(writer *zip.Writer, name string, content []byte) error {
	header := &zip.FileHeader{Name: name, Method: zip.Deflate}
	header.SetMode(0o600)
	entry, err := writer.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = entry.Write(content)
	return err
}

func validateEntryName(name string, isDir bool) error {
	if isDir {
		return fmt.Errorf("备份文件无效：不允许目录 %s", name)
	}
	if name == "" || strings.Contains(name, "\\") || strings.Contains(name, "/") || strings.Contains(name, "..") || path.IsAbs(name) || filepath.IsAbs(name) || path.Clean(name) != name {
		return fmt.Errorf("备份文件无效：包含非法路径 %s", name)
	}
	if name != databaseName && name != secretKeyName && name != manifestName {
		return fmt.Errorf("备份文件无效：包含不支持的文件 %s", name)
	}
	return nil
}

func sizeLimit(name string) (int64, bool) {
	switch name {
	case databaseName:
		return maxDatabaseBytes, true
	case secretKeyName:
		return maxSecretBytes, true
	case manifestName:
		return maxManifestBytes, true
	default:
		return 0, false
	}
}

func readZipFile(file *zip.File, limit int64) ([]byte, error) {
	if file.UncompressedSize64 > uint64(limit) {
		return nil, fmt.Errorf("备份文件无效：%s 过大", file.Name)
	}
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	var buffer bytes.Buffer
	written, err := io.Copy(&buffer, io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if written > limit {
		return nil, fmt.Errorf("备份文件无效：%s 过大", file.Name)
	}
	return buffer.Bytes(), nil
}

func validateSecretKey(content []byte) error {
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(content)))
	if err != nil {
		return errors.New("备份文件无效：secret.key 格式错误")
	}
	if len(decoded) != 32 {
		return errors.New("备份文件无效：secret.key 长度错误")
	}
	return nil
}

func validateDatabase(content []byte) error {
	if len(content) < 16 || string(content[:16]) != "SQLite format 3\x00" {
		return errors.New("备份文件无效：keymanager.sqlite 不是有效的 SQLite 数据库")
	}
	tempFile, err := os.CreateTemp("", "keymanager-validate-*.sqlite")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)
	if _, err := tempFile.Write(content); err != nil {
		tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	database, err := sql.Open("sqlite", tempPath)
	if err != nil {
		return err
	}
	defer database.Close()
	var integrity string
	if err := database.QueryRow("PRAGMA integrity_check").Scan(&integrity); err != nil {
		return err
	}
	if integrity != "ok" {
		return errors.New("备份文件无效：SQLite 完整性校验失败")
	}
	for _, table := range []string{"api_keys", "test_results", "audit_logs", "app_settings"} {
		exists, err := tableExists(database, table)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("备份文件无效：缺少数据表 %s", table)
		}
	}
	return nil
}

func tableExists(database *sql.DB, table string) (bool, error) {
	var count int
	err := database.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&count)
	return count > 0, err
}

func moveIfExists(source string, target string) (bool, error) {
	if _, err := os.Stat(source); errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	if err := os.Rename(source, target); err != nil {
		return false, err
	}
	return true, nil
}

func rollback(databasePath string, databaseBackup string, databaseMoved bool, secretPath string, secretBackup string, secretMoved bool) {
	_ = os.Remove(databasePath)
	_ = os.Remove(secretPath)
	if databaseMoved {
		_ = os.Rename(databaseBackup, databasePath)
	}
	if secretMoved {
		_ = os.Rename(secretBackup, secretPath)
	}
}
