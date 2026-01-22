#!/bin/bash
# Migration Verification Script for Video Fields
# This script verifies the database schema migration for video-specific fields

set -e

echo "======================================"
echo "Video Fields Migration Verification"
echo "======================================"
echo ""

# Test case counter
TESTS_PASSED=0
TESTS_FAILED=0

# Function to print test result
print_result() {
    if [ $1 -eq 0 ]; then
        echo "✓ PASS: $2"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo "✗ FAIL: $2"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

echo "Test Case 1: Migration Files Exist"
echo "-----------------------------------"

# Check PostgreSQL migration files
if [ -f "scripts/database/migrations/001_add_video_fields_postgres.sql" ]; then
    print_result 0 "PostgreSQL migration file exists"
else
    print_result 1 "PostgreSQL migration file missing"
fi

if [ -f "scripts/database/migrations/002_rollback_video_fields_postgres.sql" ]; then
    print_result 0 "PostgreSQL rollback file exists"
else
    print_result 1 "PostgreSQL rollback file missing"
fi

# Check MySQL migration files
if [ -f "scripts/database/migrations/001_add_video_fields_mysql.sql" ]; then
    print_result 0 "MySQL migration file exists"
else
    print_result 1 "MySQL migration file missing"
fi

if [ -f "scripts/database/migrations/002_rollback_video_fields_mysql.sql" ]; then
    print_result 0 "MySQL rollback file exists"
else
    print_result 1 "MySQL rollback file missing"
fi

echo ""
echo "Test Case 2: Migration Content Validation"
echo "------------------------------------------"

# Check PostgreSQL migration contains all required fields
POSTGRES_FILE="scripts/database/migrations/001_add_video_fields_postgres.sql"

for field in video_duration_min video_duration_max video_protocols video_start_delay video_mimes video_skippable video_skip_delay; do
    if grep -q "$field" "$POSTGRES_FILE"; then
        print_result 0 "PostgreSQL migration includes $field"
    else
        print_result 1 "PostgreSQL migration missing $field"
    fi
done

# Check MySQL migration contains all required fields
MYSQL_FILE="scripts/database/migrations/001_add_video_fields_mysql.sql"
for field in video_duration_min video_duration_max video_protocols video_start_delay video_mimes video_skippable video_skip_delay; do
    if grep -q "$field" "$MYSQL_FILE"; then
        print_result 0 "MySQL migration includes $field"
    else
        print_result 1 "MySQL migration missing $field"
    fi
done

echo ""
echo "Test Case 3: Go Struct Validation"
echo "----------------------------------"

# Check Go struct file exists
if [ -f "stored_requests/video_fields.go" ]; then
    print_result 0 "Go VideoFields struct file exists"

    # Check struct contains all fields
    GO_STRUCT_FILE="stored_requests/video_fields.go"

    for field in DurationMin DurationMax Protocols StartDelay Mimes Skippable SkipDelay; do
        if grep -q "$field" "$GO_STRUCT_FILE"; then
            print_result 0 "Go struct includes $field"
        else
            print_result 1 "Go struct missing $field"
        fi
    done

    # Check validation function exists
    if grep -q "ValidateVideoFields" "$GO_STRUCT_FILE"; then
        print_result 0 "Validation function exists"
    else
        print_result 1 "Validation function missing"
    fi
else
    print_result 1 "Go VideoFields struct file missing"
fi

echo ""
echo "Test Case 4: Go Tests Validation"
echo "---------------------------------"

# Check test file exists
if [ -f "stored_requests/video_fields_test.go" ]; then
    print_result 0 "Go test file exists"

    TEST_FILE="stored_requests/video_fields_test.go"

    # Check for test functions
    if grep -q "TestValidateVideoFields" "$TEST_FILE"; then
        print_result 0 "Main validation test exists"
    else
        print_result 1 "Main validation test missing"
    fi

    # Check for specific test cases
    for test_case in nil_fields valid_fields negative_duration_min invalid_protocol unsupported_mime_type; do
        if grep -q "$test_case" "$TEST_FILE"; then
            print_result 0 "Test case '$test_case' exists"
        else
            print_result 1 "Test case '$test_case' missing"
        fi
    done
else
    print_result 1 "Go test file missing"
fi

echo ""
echo "Test Case 5: Backwards Compatibility"
echo "-------------------------------------"

# Check PostgreSQL migration uses IF NOT EXISTS
if grep -q "IF NOT EXISTS" "$POSTGRES_FILE"; then
    print_result 0 "PostgreSQL migration uses IF NOT EXISTS"
else
    print_result 1 "PostgreSQL migration missing IF NOT EXISTS"
fi

# Check MySQL migration uses IF NOT EXISTS
if grep -q "IF NOT EXISTS" "$MYSQL_FILE"; then
    print_result 0 "MySQL migration uses IF NOT EXISTS"
else
    print_result 1 "MySQL migration missing IF NOT EXISTS"
fi

# Check for omitempty in JSON tags
if grep -q 'omitempty' "$GO_STRUCT_FILE"; then
    print_result 0 "Go struct uses omitempty for backwards compatibility"
else
    print_result 1 "Go struct missing omitempty tags"
fi

echo ""
echo "Test Case 6: Index Creation"
echo "----------------------------"

# Check PostgreSQL indexes
for index in idx_stored_requests_video_duration idx_stored_requests_video_protocols idx_stored_requests_video_mimes; do
    if grep -q "$index" "$POSTGRES_FILE"; then
        print_result 0 "PostgreSQL index '$index' exists"
    else
        print_result 1 "PostgreSQL index '$index' missing"
    fi
done

# Check MySQL indexes
for index in idx_stored_requests_video_duration idx_stored_imps_video_duration; do
    if grep -q "$index" "$MYSQL_FILE"; then
        print_result 0 "MySQL index '$index' exists"
    else
        print_result 1 "MySQL index '$index' missing"
    fi
done

echo ""
echo "Test Case 7: Documentation"
echo "--------------------------"

# Check README exists
if [ -f "scripts/database/migrations/README.md" ]; then
    print_result 0 "README.md exists"

    README_FILE="scripts/database/migrations/README.md"

    # Check README contains key sections
    if grep -q "Video Duration" "$README_FILE"; then
        print_result 0 "README documents video duration"
    else
        print_result 1 "README missing video duration documentation"
    fi

    if grep -q "Backwards Compatibility" "$README_FILE"; then
        print_result 0 "README documents backwards compatibility"
    else
        print_result 1 "README missing backwards compatibility section"
    fi
else
    print_result 1 "README.md missing"
fi

echo ""
echo "======================================"
echo "Summary"
echo "======================================"
echo "Tests Passed: $TESTS_PASSED"
echo "Tests Failed: $TESTS_FAILED"
echo ""

# Exit with appropriate code
if [ $TESTS_FAILED -gt 0 ]; then
    echo "Some tests failed. Please review the results above."
    exit 1
else
    echo "All tests passed! Migration is ready."
    exit 0
fi
