package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGetLogDetailPreservesOwnershipAndScopedAdminPermissions(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousDatabaseType := common.LogDatabaseType()
	model.DB, model.LOG_DB = db, db
	common.SetLogDatabaseType(common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.SetLogDatabaseType(previousDatabaseType)
		require.NoError(t, sqlDB.Close())
	})
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.LogDetail{}))
	require.NoError(t, db.Create(&model.User{Id: 1, Username: "detail-owner", AffCode: "owner", Role: common.RoleCommonUser}).Error)
	require.NoError(t, db.Create(&model.User{Id: 2, Username: "detail-admin", AffCode: "admin", Role: common.RoleAdminUser}).Error)
	require.NoError(t, db.Create(&model.User{Id: 3, Username: "detail-viewer", AffCode: "viewer", Role: common.RoleCommonUser}).Error)
	require.NoError(t, db.Create(&model.LogDetail{RequestId: "detail-test", UserId: 1, RequestBody: "private request"}).Error)

	for _, test := range []struct {
		name          string
		userID        int
		token, legacy bool
		scopes        []string
		success       bool
	}{
		{name: "owner browser", userID: 1, success: true},
		{name: "owner scoped token", userID: 1, token: true, scopes: []string{"usage:read"}, success: true},
		{name: "admin browser", userID: 2, success: true},
		{name: "admin legacy token", userID: 2, token: true, legacy: true, success: true},
		{name: "admin own usage scope does not expose others", userID: 2, token: true, scopes: []string{"usage:read"}},
		{name: "admin log scope", userID: 2, token: true, scopes: []string{"usage:read", "log:read"}, success: true},
		{name: "non-admin log scope cannot expose others", userID: 3, token: true, scopes: []string{"usage:read", "log:read"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(response)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/api/log/detail?request_id=detail-test", nil)
			ctx.Set("id", test.userID)
			ctx.Set("use_access_token", test.token)
			ctx.Set("access_token_legacy", test.legacy)
			ctx.Set("access_token_scopes", test.scopes)
			GetLogDetail(ctx)
			var result struct {
				Success bool `json:"success"`
			}
			require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
			assert.Equal(t, test.success, result.Success)
			if test.success {
				assert.Contains(t, response.Body.String(), "private request")
			} else {
				assert.NotContains(t, response.Body.String(), "private request")
			}
		})
	}
}
