package repository

import (
	"context"
	"testing"
	"time"
	"regexp"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"gosocial/internal/dbmysql"
)

func setupTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
	}

	return gormDB, mock, cleanup
}

func TestChatRepository_Save(t *testing.T) {
	tests := []struct {
		name        string
		message     *dbmysql.Message
		mockSetup   func(sqlmock.Sqlmock)
		expectError bool
	}{
		{
			name: "successful save",
			message: &dbmysql.Message{
				ConversationID: "conv-123",
				SenderID:       "user-456",
				Content:        "Hello, world!",
				SentAt:         time.Now().UTC(),
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				// FIXED: Include media_ref_id in expected SQL (6 parameters)
				mock.ExpectExec(regexp.QuoteMeta(
					"INSERT INTO `messages` (`conversation_id`,`sender_id`,`content`,`sent_at`,`status`,`media_ref_id`) VALUES (?,?,?,?,?,?)")).
					WithArgs("conv-123", "user-456", "Hello, world!", sqlmock.AnyArg(), "delivered", nil).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectError: false,
		},
		{
			name: "database error",
			message: &dbmysql.Message{
				ConversationID: "conv-123",
				SenderID:       "user-456",
				Content:        "Hello, world!",
				SentAt:         time.Now().UTC(),
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					"INSERT INTO `messages`")).
					WillReturnError(assert.AnError)
				mock.ExpectRollback()
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, cleanup := setupTestDB(t)
			defer cleanup()

			tt.mockSetup(mock)

			repo := NewChatRepository(db)
			err := repo.Save(context.Background(), tt.message)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestChatRepository_FetchHistory(t *testing.T) {
	tests := []struct {
		name           string
		conversationID string
		mockSetup      func(sqlmock.Sqlmock)
		expectedCount  int
		expectError    bool
	}{
		{
			name:           "successful fetch with messages",
			conversationID: "conv-123",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"message_id", "conversation_id", "sender_id", "content", "sent_at", "status", "media_ref_id",
				}).
					AddRow(1, "conv-123", "user-456", "Hello", time.Now(), "delivered", nil).
					AddRow(2, "conv-123", "user-789", "Hi there!", time.Now(), "delivered", nil)

				mock.ExpectQuery(regexp.QuoteMeta(
					"SELECT * FROM `messages` WHERE conversation_id = ?")).
					WithArgs("conv-123").
					WillReturnRows(rows)
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name:           "empty conversation",
			conversationID: "conv-empty",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"message_id", "conversation_id", "sender_id", "content", "sent_at", "status", "media_ref_id",
				})

				mock.ExpectQuery(regexp.QuoteMeta(
					"SELECT * FROM `messages` WHERE conversation_id = ?")).
					WithArgs("conv-empty").
					WillReturnRows(rows)
			},
			expectedCount: 0,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, cleanup := setupTestDB(t)
			defer cleanup()

			tt.mockSetup(mock)

			repo := NewChatRepository(db)
			messages, err := repo.FetchHistory(context.Background(), tt.conversationID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, messages, tt.expectedCount)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

