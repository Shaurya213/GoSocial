package user

import (
	"fmt"
    "context"
    "gosocial/internal/dbmysql"
    "gorm.io/gorm"
	"time"
)


type DeviceRepository interface {
    RegisterDevice(ctx context.Context, device *dbmysql.Device) error
    GetUserdDevices(ctx context.Context, userID uint64) ([]*dbmysql.Device, error)
    UpdateDeviceActivity(ctx context.Context, deviceToken string) error
    RemoveDevice(ctx context.Context, deviceToken string) error
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

func (r *deviceRepository) UpdatedDeviceActivity(ctx context.Context, deviceToken string) error {
    return r.db.WithContext(ctx).
        Model(&dbmysql.Device{}).
        Where("device_token = ?", deviceToken).
        Update("last_active", time.Now()).Error
}

func (r *deviceRepository) RemovedDevice(ctx context.Context, deviceToken string) error {
    return r.db.WithContext(ctx).Delete(&dbmysql.Device{}, "device_token = ?", deviceToken).Error
}

type deviceRepository struct {
	db *gorm.DB
}

func NewDeviceRepository(db *gorm.DB) *deviceRepository {
	return &deviceRepository{
		db: db,
	}
}

func (r *deviceRepository) CreateOrUpdate(
	ctx context.Context,
	userID, deviceToken, platform string,
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

func (r *deviceRepository) ActiveByUserID(
	ctx context.Context,
	userID string,
) ([]interface{}, error) {
	var devices []*dbmysql.Device

	cutoffTime := time.Now().AddDate(0, 0, -30)

	err := r.db.WithContext(ctx).
		Where("user_id = ? AND last_active > ?", userID, cutoffTime).
		Order("last_active DESC").
		Find(&devices).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get active devices: %w", err)
	}

	result := make([]interface{}, len(devices))
	for i, device := range devices {
		result[i] = device
	}

	return result, nil
}

func (r *deviceRepository) UpdateTokenStatus(
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

func (r *deviceRepository) DeleteToken(ctx context.Context, token string) error {
	result := r.db.WithContext(ctx).Delete(&dbmysql.Device{}, "device_token = ?", token)

	if result.Error != nil {
		return fmt.Errorf("failed to delete device token: %w", result.Error)
	}

	return nil
}
