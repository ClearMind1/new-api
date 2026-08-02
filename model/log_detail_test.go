package model

import (
	"context"
	"errors"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func setupLogDetailTestDatabases(t *testing.T) (*gorm.DB, *gorm.DB) {
	t.Helper()

	openDB := func() *gorm.DB {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
			Logger: gormlogger.Default.LogMode(gormlogger.Silent),
		})
		require.NoError(t, err)
		sqlDB, err := db.DB()
		require.NoError(t, err)
		sqlDB.SetMaxOpenConns(1)
		return db
	}

	mainDB := openDB()
	logDB := openDB()
	originalDB := DB
	originalLogDB := LOG_DB
	originalLogDatabaseType := common.LogDatabaseType()
	DB = mainDB
	LOG_DB = logDB
	t.Cleanup(func() {
		DB = originalDB
		LOG_DB = originalLogDB
		common.SetLogDatabaseType(originalLogDatabaseType)

		mainSQLDB, err := mainDB.DB()
		require.NoError(t, err)
		require.NoError(t, mainSQLDB.Close())
		logSQLDB, err := logDB.DB()
		require.NoError(t, err)
		require.NoError(t, logSQLDB.Close())
	})
	return mainDB, logDB
}

func TestLogDetailDBUsesMainDatabaseForClickHouseLogs(t *testing.T) {
	mainDB, logDB := setupLogDetailTestDatabases(t)
	common.SetLogDatabaseType(common.DatabaseTypeClickHouse)
	require.NoError(t, mainDB.AutoMigrate(&LogDetail{}))

	detail := &LogDetail{
		RequestId:    "req-clickhouse-fallback",
		RequestBody:  `{"prompt":"hello"}`,
		ResponseBody: "first response",
		CreatedAt:    10,
	}
	require.NoError(t, UpsertLogDetail(detail))

	detail.ResponseBody = "updated response"
	require.NoError(t, UpsertLogDetail(detail))

	got, err := GetLogDetailByRequestId(detail.RequestId)
	require.NoError(t, err)
	assert.Equal(t, "updated response", got.ResponseBody)

	var count int64
	require.NoError(t, mainDB.Model(&LogDetail{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
	assert.False(t, logDB.Migrator().HasTable(&LogDetail{}))

	deleted, err := DeleteOldLogDetails(context.Background(), 11, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)
	require.NoError(t, mainDB.Model(&LogDetail{}).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

func TestLogDetailDBKeepsRelationalLogDatabase(t *testing.T) {
	mainDB, logDB := setupLogDetailTestDatabases(t)
	common.SetLogDatabaseType(common.DatabaseTypeSQLite)
	require.NoError(t, logDB.AutoMigrate(&LogDetail{}))

	detail := &LogDetail{RequestId: "req-relational-log-db", CreatedAt: 10}
	require.NoError(t, UpsertLogDetail(detail))

	got, err := GetLogDetailByRequestId(detail.RequestId)
	require.NoError(t, err)
	assert.Equal(t, detail.RequestId, got.RequestId)
	assert.False(t, mainDB.Migrator().HasTable(&LogDetail{}))
}

func TestUpsertLogDetailReturnsLookupErrorsWithoutCreating(t *testing.T) {
	_, logDB := setupLogDetailTestDatabases(t)
	common.SetLogDatabaseType(common.DatabaseTypeSQLite)

	createCalled := false
	require.NoError(t, logDB.Callback().Create().Before("gorm:create").Register(
		"test:track_log_detail_create",
		func(*gorm.DB) { createCalled = true },
	))

	err := UpsertLogDetail(&LogDetail{RequestId: "req-missing-table"})
	require.Error(t, err)
	assert.False(t, createCalled)
}

func TestDeleteOldLogDetailsBatchesByIDAndHonorsCancellation(t *testing.T) {
	_, logDB := setupLogDetailTestDatabases(t)
	common.SetLogDatabaseType(common.DatabaseTypeSQLite)
	require.NoError(t, logDB.AutoMigrate(&LogDetail{}))

	details := []LogDetail{
		{RequestId: "old-1", CreatedAt: 10},
		{RequestId: "old-2", CreatedAt: 20},
		{RequestId: "old-3", CreatedAt: 30},
		{RequestId: "old-4", CreatedAt: 40},
		{RequestId: "old-5", CreatedAt: 49},
		{RequestId: "new-1", CreatedAt: 50},
	}
	require.NoError(t, logDB.Create(&details).Error)

	deleteCalls := 0
	require.NoError(t, logDB.Callback().Delete().Before("gorm:delete").Register(
		"test:count_log_detail_delete_batches",
		func(*gorm.DB) { deleteCalls++ },
	))

	deleted, err := DeleteOldLogDetails(context.Background(), 50, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(5), deleted)
	assert.Equal(t, 3, deleteCalls)

	var remaining []LogDetail
	require.NoError(t, logDB.Order("id ASC").Find(&remaining).Error)
	require.Len(t, remaining, 1)
	assert.Equal(t, "new-1", remaining[0].RequestId)

	require.NoError(t, logDB.Create(&LogDetail{RequestId: "cancelled-old", CreatedAt: 1}).Error)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	deleted, err = DeleteOldLogDetails(ctx, 50, 1)
	assert.Equal(t, int64(0), deleted)
	assert.True(t, errors.Is(err, context.Canceled))

	var cancelledOldCount int64
	require.NoError(t, logDB.Model(&LogDetail{}).
		Where("request_id = ?", "cancelled-old").
		Count(&cancelledOldCount).Error)
	assert.Equal(t, int64(1), cancelledOldCount)
}
