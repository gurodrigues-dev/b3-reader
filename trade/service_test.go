package trade_test

import (
	"context"
	"errors"
	"testing"
	"time"

	mock_reader "github.com/gurodrigues-dev/b3-reader/internal/reader/mocks"
	"github.com/gurodrigues-dev/b3-reader/trade"
	"github.com/gurodrigues-dev/b3-reader/trade/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestService_IngestFiles(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func(repo *mocks.MockRepository, csvReader *mock_reader.MockReader)
		expectedError error
	}{
		{
			name: "successfully ingests files",
			setupMocks: func(repo *mocks.MockRepository, csvReader *mock_reader.MockReader) {
				recordsChan := make(chan [][]string, 1)
				errChan := make(chan error, 1)

				recordsChan <- [][]string{
					{"header1", "header2", "header3", "header4", "header5", "header6", "header7", "header8", "header9"},
					{"1", "ABC123", "field3", "123.45", "1000", "123456", "field7", "field8", "2023-08-18"},
				}
				close(recordsChan)
				close(errChan)

				csvReader.EXPECT().Read(gomock.Any()).Return(recordsChan, errChan)

				repo.EXPECT().
					SaveBatch(gomock.Any(), gomock.Any()).
					Return(int64(1), nil).
					AnyTimes()
			},
			expectedError: nil,
		},
		{
			name: "returns error when parsing fails",
			setupMocks: func(_ *mocks.MockRepository, csvReader *mock_reader.MockReader) {
				recordsChan := make(chan [][]string, 1)
				errChan := make(chan error, 1)

				recordsChan <- [][]string{
					{"1", "ABC123", "field3", "bad-float", "1000", "123456", "field7", "field8", "2023-08-18"},
				}
				close(recordsChan)
				close(errChan)

				csvReader.EXPECT().Read(gomock.Any()).Return(recordsChan, errChan)
			},
			expectedError: nil,
		},
		{
			name: "returns error when saving batch fails",
			setupMocks: func(repo *mocks.MockRepository, csvReader *mock_reader.MockReader) {
				recordsChan := make(chan [][]string, 1)
				errChan := make(chan error, 1)

				recordsChan <- [][]string{
					{"header1", "h2", "h3", "h4", "h5", "h6", "h7", "h8", "h9"},
					{"1", "ABC123", "field3", "123.45", "1000", "123456", "field7", "field8", "2023-08-18"},
					{"2", "DEF456", "field3", "678.90", "2000", "234556", "field7", "field8", "2023-08-19"},
				}
				close(recordsChan)
				close(errChan)

				csvReader.EXPECT().Read(gomock.Any()).Return(recordsChan, errChan)

				repo.EXPECT().
					SaveBatch(gomock.Any(), gomock.Any()).
					Return(int64(0), errors.New("db error")).
					AnyTimes()
			},
			expectedError: errors.New("database save batch error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := mocks.NewMockRepository(ctrl)
			csvReader := mock_reader.NewMockReader(ctrl)
			logger := zap.NewNop()

			tt.setupMocks(repo, csvReader)

			service := trade.NewService(repo, csvReader, logger)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			err := service.IngestFiles(ctx, "test.csv")
			if tt.expectedError != nil {
				assert.ErrorContains(t, err, tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetAggregatedData(t *testing.T) {
	ctx := t.Context()

	t.Run("expect error when ticker is empty", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockRepo := mocks.NewMockRepository(ctrl)
		mockReader := mock_reader.NewMockReader(ctrl)

		svc := trade.NewService(mockRepo, mockReader, zap.NewNop())

		data, err := svc.GetAggregatedData(ctx, "", nil)

		assert.Nil(t, data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ticker is required")
	})

	t.Run("use default startdate when data_inicio is nil", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockRepo := mocks.NewMockRepository(ctrl)
		mockReader := mock_reader.NewMockReader(ctrl)

		svc := trade.NewService(mockRepo, mockReader, zap.NewNop())

		approxDate := time.Now().AddDate(0, 0, -7)

		mockRepo.
			EXPECT().
			GetAggregatedData(ctx, "PETR4", gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, startDate time.Time) (float64, int, error) {
				if !startDate.After(approxDate.Add(-2*time.Second)) || !startDate.Before(approxDate.Add(2*time.Second)) {
					t.Errorf("expected startDate ~ %v, got %v", approxDate, startDate)
				}
				return 100.5, 2000, nil
			})

		data, err := svc.GetAggregatedData(ctx, "PETR4", nil)

		assert.NoError(t, err)
		assert.Equal(t, "PETR4", data.Ticker)
		assert.Equal(t, 100.5, data.MaxRangeValue)
		assert.Equal(t, 2000, data.MaxDailyVolume)
	})

	t.Run("when repository return fail", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockRepo := mocks.NewMockRepository(ctrl)
		mockReader := mock_reader.NewMockReader(ctrl)

		svc := trade.NewService(mockRepo, mockReader, zap.NewNop())

		startDate := time.Date(2024, 8, 10, 0, 0, 0, 0, time.UTC)

		mockRepo.
			EXPECT().
			GetAggregatedData(ctx, "VALE3", startDate).
			Return(0.0, 0, errors.New("db error"))

		data, err := svc.GetAggregatedData(ctx, "VALE3", &startDate)

		assert.Nil(t, data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "fetching aggregated data error")
	})

	t.Run("successfully with data_inicio", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockRepo := mocks.NewMockRepository(ctrl)
		mockReader := mock_reader.NewMockReader(ctrl)

		svc := trade.NewService(mockRepo, mockReader, zap.NewNop())
		startDate := time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC)

		mockRepo.
			EXPECT().
			GetAggregatedData(ctx, "ITUB4", startDate).
			Return(55.5, 1200, nil)

		data, err := svc.GetAggregatedData(ctx, "ITUB4", &startDate)

		assert.NoError(t, err)
		assert.Equal(t, "ITUB4", data.Ticker)
		assert.Equal(t, 55.5, data.MaxRangeValue)
		assert.Equal(t, 1200, data.MaxDailyVolume)
	})
}
