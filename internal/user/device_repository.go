package user

import (
	"context"
	"fmt"
	"time"

	"gosocial/internal/dbmysql"

	"gorm.io/gorm"
)

type DeviceRepository interface {
	RegisterDevice(ctx context.Context, device *dbmysql.Device) error
	GetUserDevices(ctx context.Context, userID uint64) ([]*dbmysql.Device, error)
	UpdatedDeviceActivity(ctx context.Context, deviceToken string) error
	RemovedDevice(ctx context.Context, deviceToken string) error
	ActiveByUserID(ctx context.Context, userID uint64) ([]interface{}, error)
	CreateOrUpdate(ctx context.Context, userID uint64, deviceToken, platform string) error
	UpdateTokenStatus(ctx context.Context, token string, isActive bool) error
	DeleteToken(ctx context.Context, token string) error
}

type DeviceRepo struct {
	db *gorm.DB
}

/*
func NewDeviceRepository(db *gorm.DB) *DeviceRepo {
	return &DeviceRepo{db: db}
}*/

func NewDeviceRepository(db *gorm.DB) DeviceRepository {
	return &DeviceRepo{db: db}
}

func (r *DeviceRepo) RegisterDevice(ctx context.Context, device *dbmysql.Device) error {
	return r.db.WithContext(ctx).Save(device).Error
}

func (r *DeviceRepo) GetUserDevices(ctx context.Context, userID uint64) ([]*dbmysql.Device, error) {
	var devices []*dbmysql.Device
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("last_active DESC").
		Find(&devices).Error

	return devices, err
}

func (r *DeviceRepo) UpdatedDeviceActivity(ctx context.Context, deviceToken string) error {
	return r.db.WithContext(ctx).
		Model(&dbmysql.Device{}).
		Where("device_token = ?", deviceToken).
		Update("last_active", time.Now()).Error
}

func (r *DeviceRepo) RemovedDevice(ctx context.Context, deviceToken string) error {
	return r.db.WithContext(ctx).Delete(&dbmysql.Device{}, "device_token = ?", deviceToken).Error
}

func (r *DeviceRepo) CreateOrUpdate(
	ctx context.Context,
	userID uint64, deviceToken, platform string,
) error {
	device := &dbmysql.Device{
		DeviceToken:  deviceToken,
		UserID:       userID,
		Platform:     platform,
		RegisteredAt: time.Now(),
		LastActive:   time.Now(),
	}

	if err := r.db.WithContext(ctx).Save(device).Error; err != nil {
		return fmt.Errorf("failed to create/update device: %w", err)
	}

	return nil
}

func (r *DeviceRepo) ActiveByUserID(
	ctx context.Context,
	userID uint64,
) ([]interface{}, error) {
	var devices []*dbmysql.Device

	cutoffTime := time.Now().AddDate(0, 0, -30)

	err := r.db.WithContext(ctx).
		Where("user_id = ? AND last_active > ?", userID, cutoffTime).
		Order("last_active DESC").
		Find(&devices).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get active devices: %v", err)
	}

	result := make([]interface{}, len(devices))
	for i, device := range devices {
		result[i] = device
	}

	return result, nil
}

func (r *DeviceRepo) UpdateTokenStatus(
	ctx context.Context,
	token string,
	isActive bool,
) error {
	updates := map[string]interface{}{}

	if !isActive {
		updates["last_active"] = time.Now().AddDate(-1, 0, 0)
	} else {
		updates["last_active"] = time.Now()
	}

	result := r.db.WithContext(ctx).
		Model(&dbmysql.Device{}).
		Where("device_token = ?", token).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("failed to update token status: %w", result.Error)
	}

	return nil
}

func (r *DeviceRepo) DeleteToken(ctx context.Context, token string) error {
	result := r.db.WithContext(ctx).Delete(&dbmysql.Device{}, "device_token = ?", token)

	if result.Error != nil {
		return fmt.Errorf("failed to delete device token: %w", result.Error)
	}

	return nil
}
