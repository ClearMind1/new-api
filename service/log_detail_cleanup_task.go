package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"

	"github.com/bytedance/gopkg/util/gopool"
)

const (
	logDetailCleanupTickInterval = 1 * time.Hour
	logDetailCleanupBatchSize    = 500
)

var (
	logDetailCleanupOnce    sync.Once
	logDetailCleanupRunning atomic.Bool
)

// StartLogDetailCleanupTask starts a background task that periodically
// deletes old log_details records based on LogDetailRetentionDays.
func StartLogDetailCleanupTask() {
	logDetailCleanupOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		gopool.Go(func() {
			logger.LogInfo(context.Background(), "log detail cleanup task started")
			ticker := time.NewTicker(logDetailCleanupTickInterval)
			defer ticker.Stop()

			runLogDetailCleanupOnce()
			for range ticker.C {
				runLogDetailCleanupOnce()
			}
		})
	})
}

func runLogDetailCleanupOnce() {
	if !logDetailCleanupRunning.CompareAndSwap(false, true) {
		return
	}
	defer logDetailCleanupRunning.Store(false)

	retentionDays := common.LogDetailRetentionDays
	if retentionDays <= 0 {
		return
	}

	cutoff := time.Now().AddDate(0, 0, -retentionDays).Unix()
	count, err := model.DeleteOldLogDetails(cutoff, logDetailCleanupBatchSize)
	if err != nil {
		logger.LogWarn(context.Background(), fmt.Sprintf("log detail cleanup task failed: %v", err))
		return
	}
	if count > 0 {
		logger.LogInfo(context.Background(), fmt.Sprintf("log detail cleanup: deleted %d records older than %d days", count, retentionDays))
	}
}
