package model

import (
	"context"
	"errors"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

// LogDetail stores detailed request/response information for a log entry.
// Each record is keyed by request_id and contains the full request body,
// headers, and optionally the response body for debugging purposes.
type LogDetail struct {
	Id             int    `json:"id" gorm:"primaryKey;autoIncrement"`
	RequestId      string `json:"request_id" gorm:"type:varchar(64);uniqueIndex:idx_log_details_request_id;not null;default:''"`
	RequestBody    string `json:"request_body" gorm:"type:TEXT"`
	RequestPath    string `json:"request_path" gorm:"type:varchar(512);default:''"`
	RequestMethod  string `json:"request_method" gorm:"type:varchar(16);default:'POST'"`
	RequestHeaders string `json:"request_headers" gorm:"type:TEXT"`
	ResponseBody   string `json:"response_body" gorm:"type:TEXT"`
	StatusCode     int    `json:"status_code" gorm:"default:0"`
	ModelName      string `json:"model_name" gorm:"type:varchar(128);default:'';index:idx_log_details_model"`
	UserId         int    `json:"user_id" gorm:"index:idx_log_details_user"`
	CreatedAt      int64  `json:"created_at" gorm:"bigint;index:idx_log_details_created"`
}

// logDetailDB keeps request details on a relational database. ClickHouse is
// optimized for append-only usage logs and does not have a log_details table,
// while the primary database always migrates LogDetail.
func logDetailDB() *gorm.DB {
	if common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
		return DB
	}
	return LOG_DB
}

// UpsertLogDetail inserts or updates a log detail record by request_id.
func UpsertLogDetail(detail *LogDetail) error {
	db := logDetailDB()
	var existing LogDetail
	err := db.Where("request_id = ?", detail.RequestId).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(detail).Error
	}
	if err != nil {
		return err
	}
	return db.Model(&existing).Updates(map[string]interface{}{
		"request_body":    detail.RequestBody,
		"request_path":    detail.RequestPath,
		"request_method":  detail.RequestMethod,
		"request_headers": detail.RequestHeaders,
		"response_body":   detail.ResponseBody,
		"status_code":     detail.StatusCode,
		"model_name":      detail.ModelName,
		"user_id":         detail.UserId,
		"created_at":      detail.CreatedAt,
	}).Error
}

// GetLogDetailByRequestId retrieves a log detail record by request_id.
func GetLogDetailByRequestId(requestId string) (*LogDetail, error) {
	var detail LogDetail
	err := logDetailDB().Where("request_id = ?", requestId).First(&detail).Error
	if err != nil {
		return nil, err
	}
	return &detail, nil
}

// DeleteOldLogDetails deletes log details older than the target timestamp in
// bounded ID batches. Selecting IDs first keeps the delete portable across
// SQLite, MySQL, and PostgreSQL, none of which share DELETE ... LIMIT syntax.
func DeleteOldLogDetails(ctx context.Context, targetTimestamp int64, limit int) (int64, error) {
	if limit <= 0 {
		limit = 100
	}

	db := logDetailDB().WithContext(ctx)
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}

		var ids []int
		if err := db.Model(&LogDetail{}).
			Where("created_at < ?", targetTimestamp).
			Order("id ASC").
			Limit(limit).
			Pluck("id", &ids).Error; err != nil {
			return total, err
		}
		if len(ids) == 0 {
			return total, nil
		}

		result := db.Where("id IN ?", ids).Delete(&LogDetail{})
		if result.Error != nil {
			return total, result.Error
		}
		total += result.RowsAffected
	}
}

// MigrateLogDetail ensures the configured relational detail store has its table.
func MigrateLogDetail() error {
	return logDetailDB().AutoMigrate(&LogDetail{})
}
