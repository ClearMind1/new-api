package model

// LogDetail stores detailed request/response information for a log entry.
// Each record is keyed by request_id and contains the full request body,
// headers, and optionally the response body for debugging purposes.
type LogDetail struct {
	Id             int    `json:"id" gorm:"primaryKey;autoIncrement"`
	RequestId      string `json:"request_id" gorm:"type:varchar(64);uniqueIndex:idx_log_details_request_id;not null;default:''"`
	RequestBody    string `json:"request_body" gorm:"type:LONGTEXT"`
	RequestPath    string `json:"request_path" gorm:"type:varchar(512);default:''"`
	RequestMethod  string `json:"request_method" gorm:"type:varchar(16);default:'POST'"`
	RequestHeaders string `json:"request_headers" gorm:"type:TEXT"`
	ResponseBody   string `json:"response_body" gorm:"type:LONGTEXT"`
	StatusCode     int    `json:"status_code" gorm:"default:0"`
	ModelName      string `json:"model_name" gorm:"type:varchar(128);default:'';index:idx_log_details_model"`
	UserId         int    `json:"user_id" gorm:"index:idx_log_details_user"`
	CreatedAt      int64  `json:"created_at" gorm:"bigint;index:idx_log_details_created"`
}

// UpsertLogDetail inserts or updates a log detail record by request_id.
// Uses GORM's FirstOrCreate pattern for cross-database compatibility.
func UpsertLogDetail(detail *LogDetail) error {
	var existing LogDetail
	err := LOG_DB.Where("request_id = ?", detail.RequestId).First(&existing).Error
	if err != nil {
		// Not found, create new
		return LOG_DB.Create(detail).Error
	}
	// Found, update
	return LOG_DB.Model(&existing).Updates(map[string]interface{}{
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
	err := LOG_DB.Where("request_id = ?", requestId).First(&detail).Error
	if err != nil {
		return nil, err
	}
	return &detail, nil
}

// DeleteOldLogDetails deletes log details older than the target timestamp.
func DeleteOldLogDetails(targetTimestamp int64, limit int) (int64, error) {
	var total int64 = 0
	for {
		result := LOG_DB.Where("created_at < ?", targetTimestamp).Limit(limit).Delete(&LogDetail{})
		if result.Error != nil {
			return total, result.Error
		}
		total += result.RowsAffected
		if result.RowsAffected < int64(limit) {
			break
		}
	}
	return total, nil
}

func init() {
	// Register LogDetail for automatic migration in migrateLOGDB
	// This is handled by adding it to the migrateLOGDB function in main.go
}

// Ensure LogDetail is included in LOG_DB migration
func MigrateLogDetail() error {
	return LOG_DB.AutoMigrate(&LogDetail{})
}
