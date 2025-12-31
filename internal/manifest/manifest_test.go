package manifest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func TestNew(t *testing.T) {
	m := New()

	if m.Version != 1 {
		t.Errorf("Version = %d, want 1", m.Version)
	}

	if m.Files == nil {
		t.Error("Files map is nil, want initialized map")
	}

	if len(m.Files) != 0 {
		t.Errorf("Files length = %d, want 0", len(m.Files))
	}
}

func TestManifestJSONRoundtrip(t *testing.T) {
	original := &Manifest{
		Version: 1,
		Files: map[string]FileEntry{
			"project-a/session.jsonl": {
				Mtime: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
				Size:  12345,
			},
			"project-b/logs/2025-01.jsonl": {
				Mtime: time.Date(2025, 1, 2, 8, 30, 0, 0, time.UTC),
				Size:  67890,
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Manifest
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.Version != original.Version {
		t.Errorf("Version = %d, want %d", parsed.Version, original.Version)
	}

	if len(parsed.Files) != len(original.Files) {
		t.Errorf("Files count = %d, want %d", len(parsed.Files), len(original.Files))
	}

	for key, want := range original.Files {
		got, exists := parsed.Files[key]
		if !exists {
			t.Errorf("Missing file entry: %s", key)
			continue
		}
		if !got.Mtime.Equal(want.Mtime) {
			t.Errorf("File %s: Mtime = %v, want %v", key, got.Mtime, want.Mtime)
		}
		if got.Size != want.Size {
			t.Errorf("File %s: Size = %d, want %d", key, got.Size, want.Size)
		}
	}
}

func TestManifestJSONFormat(t *testing.T) {
	m := &Manifest{
		Version: 1,
		Files: map[string]FileEntry{
			"test.jsonl": {
				Mtime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				Size:  100,
			},
		},
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent failed: %v", err)
	}

	want := `{
  "version": 1,
  "files": {
    "test.jsonl": {
      "mtime": "2025-01-01T00:00:00Z",
      "size": 100
    }
  }
}`

	if string(data) != want {
		t.Errorf("JSON format mismatch:\ngot:\n%s\nwant:\n%s", data, want)
	}
}

type mockS3Client struct {
	getObjectResp *s3.GetObjectOutput
	getObjectErr  error
	putObjectResp *s3.PutObjectOutput
	putObjectErr  error
}

func (m *mockS3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return m.getObjectResp, m.getObjectErr
}

func (m *mockS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return m.putObjectResp, m.putObjectErr
}

func TestLoad_ManifestDoesNotExist(t *testing.T) {
	mock := &mockS3Client{
		getObjectErr: &types.NoSuchKey{},
	}

	m, err := Load(context.Background(), mock, "bucket", "key")
	if err != nil {
		t.Fatalf("Load failed for missing manifest: %v", err)
	}

	if m.Version != 1 {
		t.Errorf("Version = %d, want 1", m.Version)
	}

	if len(m.Files) != 0 {
		t.Errorf("Files length = %d, want 0 for new manifest", len(m.Files))
	}
}

func TestLoad_ManifestDoesNotExist_NotFound(t *testing.T) {
	mock := &mockS3Client{
		getObjectErr: &types.NotFound{},
	}

	m, err := Load(context.Background(), mock, "bucket", "key")
	if err != nil {
		t.Fatalf("Load failed for missing manifest (NotFound): %v", err)
	}

	if m.Version != 1 {
		t.Errorf("Version = %d, want 1", m.Version)
	}

	if len(m.Files) != 0 {
		t.Errorf("Files length = %d, want 0 for new manifest", len(m.Files))
	}
}

func TestLoad_Success(t *testing.T) {
	manifestJSON := `{
		"version": 1,
		"files": {
			"test.jsonl": {
				"mtime": "2025-01-01T12:00:00Z",
				"size": 12345
			}
		}
	}`

	mock := &mockS3Client{
		getObjectResp: &s3.GetObjectOutput{
			Body: io.NopCloser(bytes.NewReader([]byte(manifestJSON))),
		},
	}

	m, err := Load(context.Background(), mock, "bucket", "key")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if m.Version != 1 {
		t.Errorf("Version = %d, want 1", m.Version)
	}

	if len(m.Files) != 1 {
		t.Fatalf("Files length = %d, want 1", len(m.Files))
	}

	entry, exists := m.Files["test.jsonl"]
	if !exists {
		t.Fatal("Expected file entry 'test.jsonl' not found")
	}

	if entry.Size != 12345 {
		t.Errorf("Size = %d, want 12345", entry.Size)
	}

	expectedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	if !entry.Mtime.Equal(expectedTime) {
		t.Errorf("Mtime = %v, want %v", entry.Mtime, expectedTime)
	}
}

func TestLoad_CorruptJSON(t *testing.T) {
	mock := &mockS3Client{
		getObjectResp: &s3.GetObjectOutput{
			Body: io.NopCloser(bytes.NewReader([]byte("not valid json"))),
		},
	}

	_, err := Load(context.Background(), mock, "bucket", "key")
	if err == nil {
		t.Fatal("Expected error for corrupt JSON, got nil")
	}
}

func TestLoad_UnsupportedVersion(t *testing.T) {
	manifestJSON := `{
		"version": 999,
		"files": {}
	}`

	mock := &mockS3Client{
		getObjectResp: &s3.GetObjectOutput{
			Body: io.NopCloser(bytes.NewReader([]byte(manifestJSON))),
		},
	}

	_, err := Load(context.Background(), mock, "bucket", "key")
	if err == nil {
		t.Fatal("Expected error for unsupported version, got nil")
	}
}

func TestLoad_NetworkError(t *testing.T) {
	mock := &mockS3Client{
		getObjectErr: errors.New("network timeout"),
	}

	_, err := Load(context.Background(), mock, "bucket", "key")
	if err == nil {
		t.Fatal("Expected error for network failure, got nil")
	}
}

func TestLoad_NilFilesMap(t *testing.T) {
	manifestJSON := `{
		"version": 1,
		"files": null
	}`

	mock := &mockS3Client{
		getObjectResp: &s3.GetObjectOutput{
			Body: io.NopCloser(bytes.NewReader([]byte(manifestJSON))),
		},
	}

	m, err := Load(context.Background(), mock, "bucket", "key")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if m.Files == nil {
		t.Error("Files map is nil, expected initialized map")
	}

	if len(m.Files) != 0 {
		t.Errorf("Files length = %d, want 0", len(m.Files))
	}
}

func TestSave_Success(t *testing.T) {
	m := &Manifest{
		Version: 1,
		Files: map[string]FileEntry{
			"test.jsonl": {
				Mtime: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
				Size:  12345,
			},
		},
	}

	mock := &mockS3Client{
		putObjectResp: &s3.PutObjectOutput{},
	}

	err := Save(context.Background(), mock, "bucket", "key", m)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
}

func TestSave_NetworkError(t *testing.T) {
	m := New()

	mock := &mockS3Client{
		putObjectErr: errors.New("network timeout"),
	}

	err := Save(context.Background(), mock, "bucket", "key", m)
	if err == nil {
		t.Fatal("Expected error for network failure, got nil")
	}
}
