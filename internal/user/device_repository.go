package user

import (
	"GoSocial/internal/dbmysql"
	"context"
	"gorm.io/gorm"
	"time"
)

type DeviceRepository interface {
	RegisterDevice(ctx context.Context, device *dbmysql.Device) error
	GetUserDevices(ctx context.Context, userID uint64) ([]*dbmysql.Device, error)
	UpdateDeviceActivity(ctx context.Context, deviceToken string) error
	RemoveDevice(ctx context.Context, deviceToken string) error
}

type deviceRepository struct {
	db *gorm.DB
}

func NewDeviceRepository(db *gorm.DB) DeviceRepository {
	return &deviceRepository{db: db}
}

func (r *deviceRepository) RegisterDevice(ctx context.Context, device *dbmysql.Device) error {
	return r.db.WithContext(ctx).Save(device).Error
}

func (r *deviceRepository) GetUserDevices(ctx context.Context, userID uint64) ([]*dbmysql.Device, error) {
	var devices []*dbmysql.Device
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("last_active DESC").
		Find(&devices).Error

	return devices, err
}

func (r *deviceRepository) UpdateDeviceActivity(ctx context.Context, deviceToken string) error {
	return r.db.WithContext(ctx).
		Model(&dbmysql.Device{}).
		Where("device_token = ?", deviceToken).
		Update("last_active", time.Now()).Error
}

func (r *deviceRepository) RemoveDevice(ctx context.Context, deviceToken string) error {
	return r.db.WithContext(ctx).Delete(&dbmysql.Device{}, "device_token = ?", deviceToken).Error
}
