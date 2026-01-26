package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

// TestBidderStore_Update_OptimisticLocking_Success tests successful update with correct version
func TestBidderStore_Update_OptimisticLocking_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()

	store := NewBidderStore(db)
	ctx := context.Background()

	bidder := createTestBidder("appnexus")
	bidder.Version = 1
	bidder.BidderName = "Updated Name"

	// Expect transaction begin
	mock.ExpectBegin()

	// Expect version check query
	versionRows := sqlmock.NewRows([]string{"version"}).AddRow(1)
	mock.ExpectQuery("SELECT version FROM bidders WHERE bidder_code").
		WithArgs("appnexus").
		WillReturnRows(versionRows)

	// Expect update query with version check
	mock.ExpectExec("UPDATE bidders").
		WithArgs(
			bidder.BidderName,
			bidder.EndpointURL,
			bidder.TimeoutMs,
			bidder.Enabled,
			bidder.Status,
			bidder.SupportsBanner,
			bidder.SupportsVideo,
			bidder.SupportsNative,
			bidder.SupportsAudio,
			bidder.GVLVendorID,
			sqlmock.AnyArg(), // http_headers JSON
			bidder.Description,
			bidder.DocumentationURL,
			bidder.ContactEmail,
			bidder.BidderCode,
			1, // version
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Expect commit
	mock.ExpectCommit()

	err = store.Update(ctx, bidder)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if bidder.Version != 2 {
		t.Errorf("Expected version to be incremented to 2, got %d", bidder.Version)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// TestBidderStore_Update_OptimisticLocking_VersionMismatch tests concurrent modification detection
func TestBidderStore_Update_OptimisticLocking_VersionMismatch(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()

	store := NewBidderStore(db)
	ctx := context.Background()

	bidder := createTestBidder("appnexus")
	bidder.Version = 1
	bidder.BidderName = "Updated Name"

	// Expect transaction begin
	mock.ExpectBegin()

	// Expect version check query - but return different version (simulating concurrent update)
	versionRows := sqlmock.NewRows([]string{"version"}).AddRow(2) // Version changed!
	mock.ExpectQuery("SELECT version FROM bidders WHERE bidder_code").
		WithArgs("appnexus").
		WillReturnRows(versionRows)

	// Expect rollback due to version mismatch
	mock.ExpectRollback()

	err = store.Update(ctx, bidder)
	if err == nil {
		t.Fatal("Expected error for version mismatch, got nil")
	}

	if !contains(err.Error(), "concurrent modification detected") {
		t.Errorf("Expected concurrent modification error, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// TestBidderStore_Update_OptimisticLocking_NotFound tests update of non-existent bidder
func TestBidderStore_Update_OptimisticLocking_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()

	store := NewBidderStore(db)
	ctx := context.Background()

	bidder := createTestBidder("nonexistent")
	bidder.Version = 1

	// Expect transaction begin
	mock.ExpectBegin()

	// Expect version check query - return no rows
	mock.ExpectQuery("SELECT version FROM bidders WHERE bidder_code").
		WithArgs("nonexistent").
		WillReturnError(sql.ErrNoRows)

	// Expect rollback
	mock.ExpectRollback()

	err = store.Update(ctx, bidder)
	if err == nil {
		t.Fatal("Expected error for non-existent bidder, got nil")
	}

	if !contains(err.Error(), "not found") {
		t.Errorf("Expected not found error, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// TestBidderStore_Update_OptimisticLocking_ZeroRowsAffected tests when update affects 0 rows
func TestBidderStore_Update_OptimisticLocking_ZeroRowsAffected(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()

	store := NewBidderStore(db)
	ctx := context.Background()

	bidder := createTestBidder("appnexus")
	bidder.Version = 1

	// Expect transaction begin
	mock.ExpectBegin()

	// Expect version check - returns matching version
	versionRows := sqlmock.NewRows([]string{"version"}).AddRow(1)
	mock.ExpectQuery("SELECT version FROM bidders WHERE bidder_code").
		WithArgs("appnexus").
		WillReturnRows(versionRows)

	// Expect update query - but return 0 rows affected (race condition)
	mock.ExpectExec("UPDATE bidders").
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Expect rollback
	mock.ExpectRollback()

	err = store.Update(ctx, bidder)
	if err == nil {
		t.Fatal("Expected error for zero rows affected, got nil")
	}

	if !contains(err.Error(), "concurrent modification detected") {
		t.Errorf("Expected concurrent modification error, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// TestBidderStore_Update_OptimisticLocking_TransactionError tests transaction errors
func TestBidderStore_Update_OptimisticLocking_TransactionError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()

	store := NewBidderStore(db)
	ctx := context.Background()

	bidder := createTestBidder("appnexus")
	bidder.Version = 1

	// Expect transaction begin to fail
	mock.ExpectBegin().WillReturnError(errors.New("transaction error"))

	err = store.Update(ctx, bidder)
	if err == nil {
		t.Fatal("Expected error from transaction begin, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// TestBidderStore_Update_OptimisticLocking_CommitError tests commit failures
func TestBidderStore_Update_OptimisticLocking_CommitError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()

	store := NewBidderStore(db)
	ctx := context.Background()

	bidder := createTestBidder("appnexus")
	bidder.Version = 1

	// Expect transaction begin
	mock.ExpectBegin()

	// Expect version check
	versionRows := sqlmock.NewRows([]string{"version"}).AddRow(1)
	mock.ExpectQuery("SELECT version FROM bidders WHERE bidder_code").
		WithArgs("appnexus").
		WillReturnRows(versionRows)

	// Expect update
	mock.ExpectExec("UPDATE bidders").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Expect commit to fail
	mock.ExpectCommit().WillReturnError(errors.New("commit failed"))

	err = store.Update(ctx, bidder)
	if err == nil {
		t.Fatal("Expected error from commit, got nil")
	}

	if !contains(err.Error(), "commit") {
		t.Errorf("Expected commit error, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// TestBidderStore_GetByCode_IncludesVersion tests that version is retrieved
func TestBidderStore_GetByCode_IncludesVersion(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()

	store := NewBidderStore(db)
	ctx := context.Background()

	expectedBidder := createTestBidder("appnexus")
	expectedBidder.Version = 5
	httpHeadersJSON, _ := json.Marshal(expectedBidder.HTTPHeaders)

	rows := sqlmock.NewRows([]string{
		"id", "bidder_code", "bidder_name", "endpoint_url", "timeout_ms",
		"enabled", "status", "supports_banner", "supports_video", "supports_native", "supports_audio",
		"gvl_vendor_id", "http_headers", "description", "documentation_url", "contact_email",
		"version", "created_at", "updated_at",
	}).AddRow(
		expectedBidder.ID,
		expectedBidder.BidderCode,
		expectedBidder.BidderName,
		expectedBidder.EndpointURL,
		expectedBidder.TimeoutMs,
		expectedBidder.Enabled,
		expectedBidder.Status,
		expectedBidder.SupportsBanner,
		expectedBidder.SupportsVideo,
		expectedBidder.SupportsNative,
		expectedBidder.SupportsAudio,
		expectedBidder.GVLVendorID,
		httpHeadersJSON,
		expectedBidder.Description,
		expectedBidder.DocumentationURL,
		expectedBidder.ContactEmail,
		expectedBidder.Version,
		expectedBidder.CreatedAt,
		expectedBidder.UpdatedAt,
	)

	mock.ExpectQuery("SELECT (.+) FROM bidders WHERE bidder_code").
		WithArgs("appnexus").
		WillReturnRows(rows)

	bidder, err := store.GetByCode(ctx, "appnexus")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if bidder.Version != 5 {
		t.Errorf("Expected version 5, got %d", bidder.Version)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// TestBidderStore_Create_ReturnsVersion tests that create returns initial version
func TestBidderStore_Create_ReturnsVersion(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()

	store := NewBidderStore(db)
	ctx := context.Background()

	bidder := createTestBidder("newbidder")
	bidder.ID = ""

	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "version", "created_at", "updated_at"}).
		AddRow("123", 1, now, now)

	mock.ExpectQuery("INSERT INTO bidders").
		WillReturnRows(rows)

	err = store.Create(ctx, bidder)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if bidder.Version != 1 {
		t.Errorf("Expected version 1, got %d", bidder.Version)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
