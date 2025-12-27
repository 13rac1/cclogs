// Package types defines the core data structures used throughout ccls.
// This includes configuration structs, project metadata, and shared types.
package types

// Config represents the complete configuration for ccls.
type Config struct {
	Local LocalConfig `yaml:"local"`
	S3    S3Config    `yaml:"s3"`
	Auth  AuthConfig  `yaml:"auth"`
}

// LocalConfig holds local filesystem settings.
type LocalConfig struct {
	ProjectsRoot string `yaml:"projects_root"`
}

// S3Config holds S3-compatible storage settings.
type S3Config struct {
	Bucket         string `yaml:"bucket"`
	Prefix         string `yaml:"prefix"`
	Region         string `yaml:"region"`
	Endpoint       string `yaml:"endpoint"`
	ForcePathStyle bool   `yaml:"force_path_style"`
}

// AuthConfig holds authentication credentials.
type AuthConfig struct {
	Profile         string `yaml:"profile"`
	AccessKeyID     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
	SessionToken    string `yaml:"session_token"`
}

// Project represents a local or remote project with JSONL file counts.
type Project struct {
	Name        string
	LocalPath   string
	LocalCount  int
	RemotePath  string
	RemoteCount int
}
