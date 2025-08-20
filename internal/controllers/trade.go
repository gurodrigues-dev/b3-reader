package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gurodrigues-dev/b3-reader/trade"
	"go.uber.org/zap"
)

type Controller struct {
	service trade.Usecase
	logger  *zap.Logger
}

func NewController(s trade.Usecase, l *zap.Logger) *Controller {
	return &Controller{
		service: s,
		logger:  l,
	}
}

// GetTrade godoc
// @Summary      Obtém dados agregados de negociações
// @Description  Retorna dados agregados de um ticker específico, podendo filtrar por data de início
// @Tags         trade
// @Accept       json
// @Produce      json
// @Param        ticker       query     string  true  "Código do ticker (ex: PETR4)"
// @Param        data_inicio  query     string  false "Data de início no formato YYYY-MM-DD"
// @Success      200          {object}  trade.AggregatedData
// @Failure      400          {object}  object
// @Failure      500          {object}  object
// @Router       /trades [get]
func (ctrl *Controller) GetTrade(ctx *gin.Context) {
	ticker := ctx.Query("ticker")
	if ticker == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ticker is required"})
		return
	}
	ctrl.logger.Info("ticker recognized", zap.String("ticker", ticker))

	dataInicio := ctx.Query("data_inicio")
	var startDate *time.Time

	if dataInicio != "" {
		parsedDate, err := time.Parse("2006-01-02", dataInicio)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid data_inicio format"})
			return
		}
		startDate = &parsedDate
	}

	ctrl.logger.Info("getting aggregated data")
	result, err := ctrl.service.GetAggregatedData(ctx.Request.Context(), ticker, startDate)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, result)
}
