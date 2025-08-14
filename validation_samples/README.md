# Email Processing Validation Samples

This directory contains sample email inputs and expected RSS output for validating the EmailRSS processing pipeline.

## File Format

### Input Files (.in)
Input files contain JSON with the following structure:
```json
{
  "uid": 12345,
  "subject": "Email Subject",
  "from": "sender@example.com", 
  "date": "2025-08-01T09:00:00Z",
  "text_body": "Plain text content...",
  "html_body": "HTML content..."
}
```

**Required fields:**
- `uid`: Unique message ID (uint32)
- `subject`: Email subject line
- `from`: Sender email address
- `date`: ISO 8601 date string (RFC3339)
- At least one of `text_body` or `html_body` must be non-empty

### Output Files (.out)
Output files contain the expected RSS item description content after processing through the EmailRSS pipeline.

## Current Test Cases

1. **plain_text_email**: Basic plain text email processing
2. **html_newsletter**: HTML email with rich formatting
3. **mime_with_boundaries**: Complex MIME multipart with quoted-printable encoding

## Running Validation Tests

```bash
# Run all validation tests
go test ./internal/processor -run "TestValidationSamples" -v

# Run format validation tests
go test ./internal/processor -run "TestValidationSampleFormat" -v
```

## Adding New Test Cases

1. Create a new `.in` file with sample email data
2. Run the test to see the actual output:
   ```bash
   go test ./internal/processor -run "TestValidationSamples/your_test_name" -v
   ```
3. Create a corresponding `.out` file with the expected output
4. Re-run the test to verify it passes

## What Gets Tested

The validation system tests the complete email processing pipeline:
- MIME boundary cleaning
- Quoted-printable decoding
- UTF-8 encoding fixes
- HTML vs text content separation
- XML entity encoding
- Content truncation and formatting

This ensures that changes to the processing logic don't break existing functionality and that the RSS output remains consistent.