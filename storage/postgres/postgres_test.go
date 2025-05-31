package postgres

import (
	"context"
	"testing"

	"github.com/and161185/metrics-alerting/model"
	"github.com/and161185/metrics-alerting/storage/postgres/mocks"
	"github.com/golang/mock/gomock"
)

func TestMockStorage_Save(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockStorage(ctrl)

	metric := &model.Metric{ID: "test", Type: model.Gauge}

	mockStorage.EXPECT().
		Save(gomock.Any(), metric).
		Return(nil)

	err := mockStorage.Save(context.Background(), metric)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMockStorage_Get(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockStorage(ctrl)

	metric := &model.Metric{ID: "test", Type: model.Gauge}

	mockStorage.EXPECT().
		Get(gomock.Any(), metric).
		Return(nil, nil)

	_, err := mockStorage.Get(context.Background(), metric)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMockStorage_GetAll(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockStorage(ctrl)

	mockStorage.EXPECT().
		GetAll(gomock.Any()).
		Return(nil, nil)

	_, err := mockStorage.GetAll(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMockStorage_SaveToFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockStorage(ctrl)

	filePath := "/tmp/test.json"

	mockStorage.EXPECT().
		SaveToFile(gomock.Any(), filePath).
		Return(nil)

	err := mockStorage.SaveToFile(context.Background(), filePath)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMockStorage_LoadFromFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockStorage(ctrl)

	filePath := "/tmp/test.json"

	mockStorage.EXPECT().
		LoadFromFile(gomock.Any(), filePath).
		Return(nil)

	err := mockStorage.LoadFromFile(context.Background(), filePath)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMockStorage_Ping(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockStorage(ctrl)

	mockStorage.EXPECT().
		Ping(gomock.Any()).
		Return(nil)

	err := mockStorage.Ping(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
