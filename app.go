package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"keymanager/internal/backup"
	"keymanager/internal/crypto"
	"keymanager/internal/db"
	"keymanager/internal/model"
	"keymanager/internal/repository"
	"keymanager/internal/service"
	"keymanager/internal/storage"
	"keymanager/internal/tester"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx           context.Context
	db            *sql.DB
	service       *service.KeyService
	maintenanceMu sync.Mutex
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if err := a.openResources(); err != nil {
		panic(err)
	}
}

func (a *App) shutdown(ctx context.Context) {
	a.closeResources()
}

func (a *App) openResources() error {
	database, err := db.Open()
	if err != nil {
		return err
	}
	secretStore, err := crypto.NewSecretStore()
	if err != nil {
		_ = database.Close()
		return err
	}
	a.db = database
	a.service = service.NewKeyService(
		repository.NewAPIKeyRepository(database),
		repository.NewTestResultRepository(database),
		repository.NewAuditRepository(database),
		secretStore,
		tester.NewModelTester(),
	)
	return nil
}

func (a *App) closeResources() {
	if a.db != nil {
		_ = a.db.Close()
	}
	a.db = nil
	a.service = nil
}

func (a *App) ListAPIKeys(filter model.APIKeyFilter) (model.APIKeyListResult, error) {
	return a.service.ListAPIKeys(filter)
}

func (a *App) GetAPIKeyDetail(id int64) (*model.APIKeyDetail, error) {
	return a.service.GetAPIKeyDetail(id)
}

func (a *App) CreateAPIKey(input model.CreateAPIKeyInput) (*model.APIKey, error) {
	return a.service.CreateAPIKey(input)
}

func (a *App) BatchImportAPIKeys(input model.BatchImportAPIKeysInput) (model.BatchImportAPIKeysResult, error) {
	return a.service.BatchImportAPIKeys(input)
}

func (a *App) BatchDeleteAPIKeys(input model.BatchDeleteAPIKeysInput) (model.BatchDeleteAPIKeysResult, error) {
	return a.service.BatchDeleteAPIKeys(input)
}

func (a *App) BatchTestAPIKeys(input model.BatchTestAPIKeysInput) (model.BatchTestAPIKeysResult, error) {
	return a.service.BatchTestAPIKeys(a.ctx, input)
}

func (a *App) BatchExportAPIKeys(input model.BatchExportAPIKeysInput) (model.BatchExportAPIKeysResult, error) {
	result := model.BatchExportAPIKeysResult{Total: len(input.IDs)}
	if len(input.IDs) == 0 {
		return result, errors.New("请选择要导出的 API Key")
	}
	path, err := wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		Title:           "导出 API Key 为 TXT",
		DefaultFilename: "api-keys-" + time.Now().Format("20060102-150405") + ".txt",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "Text Files (*.txt)", Pattern: "*.txt"},
		},
	})
	if err != nil {
		return result, err
	}
	if strings.TrimSpace(path) == "" {
		result.Canceled = true
		return result, nil
	}
	if !strings.EqualFold(filepath.Ext(path), ".txt") {
		path += ".txt"
	}
	export, err := a.service.BuildAPIKeysTXTExport(input)
	if err != nil {
		return result, err
	}
	if err := os.WriteFile(path, []byte(export.Content()), 0600); err != nil {
		return result, err
	}
	result.Exported = export.Exported()
	result.Skipped = export.Skipped()
	result.Path = path
	a.service.AuditAPIKeysTXTExport(result.Total, result.Exported, result.Skipped)
	return result, nil
}

func (a *App) BackupData() (model.BackupRestoreResult, error) {
	result := model.BackupRestoreResult{}
	path, err := wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		Title:           "备份 KeyManager 数据",
		DefaultFilename: "keymanager-backup-" + time.Now().Format("20060102-150405") + ".zip",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "Zip Files (*.zip)", Pattern: "*.zip"},
		},
	})
	if err != nil {
		return result, err
	}
	if strings.TrimSpace(path) == "" {
		result.Canceled = true
		return result, nil
	}
	if !strings.EqualFold(filepath.Ext(path), ".zip") {
		path += ".zip"
	}

	a.maintenanceMu.Lock()
	defer a.maintenanceMu.Unlock()

	paths, err := storage.AppPaths()
	if err != nil {
		return result, err
	}
	if err := backup.CreateBackup(path, paths); err != nil {
		return result, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return result, err
	}
	result.Path = path
	a.service.AuditDataBackup(path, info.Size())
	return result, nil
}

func (a *App) RestoreData() (model.BackupRestoreResult, error) {
	result := model.BackupRestoreResult{}
	path, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "选择 KeyManager 备份文件",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "Zip Files (*.zip)", Pattern: "*.zip"},
		},
	})
	if err != nil {
		return result, err
	}
	if strings.TrimSpace(path) == "" {
		result.Canceled = true
		return result, nil
	}
	validated, err := backup.ValidateBackup(path)
	if err != nil {
		return result, err
	}

	a.maintenanceMu.Lock()
	defer a.maintenanceMu.Unlock()

	paths, err := storage.AppPaths()
	if err != nil {
		return result, err
	}
	a.closeResources()
	if err := backup.RestoreBackup(validated, paths); err != nil {
		if openErr := a.openResources(); openErr != nil {
			return result, fmt.Errorf("还原失败，且重新打开原数据失败: %v; %w", openErr, err)
		}
		return result, err
	}
	if err := a.openResources(); err != nil {
		backup.RollbackRestore(paths)
		if openErr := a.openResources(); openErr != nil {
			return result, fmt.Errorf("还原失败，且恢复原数据后重新打开失败: %v; %w", openErr, err)
		}
		return result, fmt.Errorf("还原失败，已恢复原数据: %w", err)
	}
	backup.CommitRestore(paths)
	result.Path = path
	a.service.AuditDataRestore(path)
	return result, nil
}

func (a *App) UpdateAPIKey(id int64, input model.UpdateAPIKeyInput) (*model.APIKey, error) {
	return a.service.UpdateAPIKey(id, input)
}

func (a *App) DisableAPIKey(id int64) (*model.APIKey, error) {
	return a.service.DisableAPIKey(id)
}

func (a *App) EnableAPIKey(id int64) (*model.APIKey, error) {
	return a.service.EnableAPIKey(id)
}

func (a *App) TestAPIKey(id int64) (*model.APIKeyDetail, error) {
	return a.service.TestAPIKey(a.ctx, id)
}

func (a *App) GetDashboardStats() (model.DashboardStats, error) {
	return a.service.GetDashboardStats()
}

func (a *App) GetFilterOptions() (model.FilterOptions, error) {
	return a.service.GetFilterOptions()
}

func (a *App) ListAuditLogs(limit int) ([]model.AuditLog, error) {
	return a.service.ListAuditLogs(limit)
}

func (a *App) CopyBaseURLAudit(id int64) error {
	return a.service.CopyBaseURLAudit(id)
}

func (a *App) GetCopyCredentials(id int64) (string, error) {
	return a.service.GetCopyCredentials(id)
}
