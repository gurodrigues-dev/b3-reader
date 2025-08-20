package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gurodrigues-dev/b3-reader/trade"
	"github.com/gurodrigues-dev/b3-reader/trade/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func setupRouter(ctrl *Controller) *gin.Engine {
	r := gin.New()
	r.GET("/trade", ctrl.GetTrade)
	return r
}

func TestController_GetTrade(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("ticker return 400", func(t *testing.T) {
		ctx := t.Context()
		ctrl := NewController(nil, zap.NewNop())
		router := setupRouter(ctrl)

		req, _ := http.NewRequestWithContext(ctx, "GET", "/trade", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "ticker is required")
	})

	t.Run("invalid data_inicio return 400", func(t *testing.T) {
		ctx := t.Context()
		ctrl := NewController(nil, zap.NewNop())
		router := setupRouter(ctrl)

		req, _ := http.NewRequestWithContext(ctx, "GET", "/trade?ticker=PETR4&data_inicio=16-08-2024", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid data_inicio format")
	})

	t.Run("service error, return 500", func(t *testing.T) {
		ctx := t.Context()
		ctrlMock := gomock.NewController(t)
		defer ctrlMock.Finish()
		mockSvc := mocks.NewMockUsecase(ctrlMock)

		ctrl := NewController(mockSvc, zap.NewNop())
		router := setupRouter(ctrl)

		startDate := time.Date(2024, 8, 16, 0, 0, 0, 0, time.UTC)

		mockSvc.
			EXPECT().
			GetAggregatedData(gomock.Any(), "VALE3", &startDate).
			Return(nil, assert.AnError)

		req, _ := http.NewRequestWithContext(ctx, "GET", "/trade?ticker=VALE3&data_inicio=2024-08-16", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "assert.AnError")
	})

	t.Run("successfully call, return 200", func(t *testing.T) {
		ctx := t.Context()
		ctrlMock := gomock.NewController(t)
		defer ctrlMock.Finish()
		mockSvc := mocks.NewMockUsecase(ctrlMock)

		ctrl := NewController(mockSvc, zap.NewNop())
		router := setupRouter(ctrl)

		mockSvc.
			EXPECT().
			GetAggregatedData(gomock.Any(), "ITUB4", gomock.Nil()).
			Return(&trade.AggregatedData{
				Ticker:         "ITUB4",
				MaxRangeValue:  12.34,
				MaxDailyVolume: 500,
			}, nil)

		req, _ := http.NewRequestWithContext(ctx, "GET", "/trade?ticker=ITUB4", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		assert.Contains(t, w.Body.String(), `"ticker":"ITUB4"`)
		assert.Contains(t, w.Body.String(), `"max_range_value":12.34`)
		assert.Contains(t, w.Body.String(), `"max_daily_volume":500`)

		var resp trade.AggregatedData
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "ITUB4", resp.Ticker)
		assert.Equal(t, 12.34, resp.MaxRangeValue)
		assert.Equal(t, 500, resp.MaxDailyVolume)
	})
}
