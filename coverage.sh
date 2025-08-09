#!/bin/bash

# Run tests with coverage for main packages (excluding slow IMAP tests)
go test ./internal/config ./internal/db ./internal/rss ./internal/server ./internal/processor -coverprofile=coverage.out

# Generate coverage summary
echo "=== Coverage Summary ==="
go tool cover -func=coverage.out | grep total

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html
echo "HTML report generated: coverage.html"

# Check if we reached 80% target
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
echo "Current coverage: ${COVERAGE}%"

if (( $(echo "$COVERAGE >= 80.0" | bc -l) )); then
    echo "✅ Coverage target of 80% achieved!"
    exit 0
else
    echo "❌ Coverage below 80% target. Current: ${COVERAGE}%"
    exit 1
fi