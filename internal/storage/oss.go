package storage

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/xid"

	"github.com/syt3s/TreeBox/internal/config"
)

const maxImageSize = 10 << 20

type UploadKind string

const (
	UploadKindAvatar     UploadKind = "avatar"
	UploadKindBackground UploadKind = "background"
)

var (
	ErrUploadNotConfigured = errors.New("upload service not configured")
	ErrEmptyFile           = errors.New("empty file")
	ErrFileTooLarge        = errors.New("file too large")
	ErrUnsupportedImage    = errors.New("unsupported image type")
)

type bucketConfig struct {
	UploadScheme string
	PublicScheme string
	Endpoint     string
	AccessID     string
	AccessSecret string
	Bucket       string
	CDNHost      string
}

var imageExtensions = map[string]string{
	"image/gif":  ".gif",
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

func UploadUserImage(ctx context.Context, userUID string, kind UploadKind, header *multipart.FileHeader) (string, error) {
	cfg, err := resolveBucketConfig()
	if err != nil {
		return "", err
	}

	file, err := header.Open()
	if err != nil {
		return "", errors.Wrap(err, "open upload file")
	}
	defer func() {
		_ = file.Close()
	}()

	content, contentType, extension, err := readImage(file)
	if err != nil {
		return "", err
	}

	objectKey := path.Join("users", safePathSegment(userUID), string(kind), xid.New().String()+extension)
	if err := cfg.upload(ctx, objectKey, content, contentType); err != nil {
		return "", err
	}

	return cfg.publicURL(objectKey), nil
}

func readImage(file multipart.File) ([]byte, string, string, error) {
	limited := io.LimitReader(file, maxImageSize+1)
	content, err := io.ReadAll(limited)
	if err != nil {
		return nil, "", "", errors.Wrap(err, "read upload file")
	}
	if len(content) == 0 {
		return nil, "", "", ErrEmptyFile
	}
	if len(content) > maxImageSize {
		return nil, "", "", ErrFileTooLarge
	}

	contentType := http.DetectContentType(content)
	extension, ok := imageExtensions[contentType]
	if !ok {
		return nil, "", "", ErrUnsupportedImage
	}

	return content, contentType, extension, nil
}

func resolveBucketConfig() (bucketConfig, error) {
	imageConfig := bucketConfig{
		UploadScheme: "https",
		PublicScheme: "https",
		Endpoint:     normalizeHost(config.Upload.ImageEndpoint),
		AccessID:     strings.TrimSpace(config.Upload.ImageAccessID),
		AccessSecret: strings.TrimSpace(config.Upload.ImageAccessSecret),
		Bucket:       strings.TrimSpace(config.Upload.ImageBucket),
		CDNHost:      normalizeHost(config.Upload.ImageBucketCDNHost),
	}
	if imageConfig.complete() {
		imageConfig.UploadScheme = endpointScheme(config.Upload.ImageEndpoint)
		if imageConfig.CDNHost != "" {
			imageConfig.PublicScheme = endpointScheme(config.Upload.ImageBucketCDNHost)
		}
		return imageConfig, nil
	}

	aliyunConfig := bucketConfig{
		UploadScheme: endpointScheme(config.Upload.AliyunEndpoint),
		PublicScheme: endpointScheme(config.Upload.AliyunEndpoint),
		Endpoint:     normalizeHost(config.Upload.AliyunEndpoint),
		AccessID:     strings.TrimSpace(config.Upload.AliyunAccessID),
		AccessSecret: strings.TrimSpace(config.Upload.AliyunAccessSecret),
		Bucket:       strings.TrimSpace(config.Upload.AliyunBucket),
		CDNHost:      normalizeHost(config.Upload.AliyunBucketCDNHost),
	}
	if aliyunConfig.complete() {
		if aliyunConfig.CDNHost != "" {
			aliyunConfig.PublicScheme = endpointScheme(config.Upload.AliyunBucketCDNHost)
		}
		return aliyunConfig, nil
	}

	return bucketConfig{}, ErrUploadNotConfigured
}

func (cfg bucketConfig) complete() bool {
	return cfg.Endpoint != "" && cfg.AccessID != "" && cfg.AccessSecret != "" && cfg.Bucket != ""
}

func (cfg bucketConfig) upload(ctx context.Context, objectKey string, content []byte, contentType string) error {
	uploadURL, host := cfg.objectURL(objectKey)
	sum := md5.Sum(content)
	contentMD5 := base64.StdEncoding.EncodeToString(sum[:])
	date := time.Now().UTC().Format(http.TimeFormat)
	canonicalResource := "/" + cfg.Bucket + "/" + objectKey

	stringToSign := strings.Join([]string{
		http.MethodPut,
		contentMD5,
		contentType,
		date,
		canonicalResource,
	}, "\n")

	mac := hmac.New(sha1.New, []byte(cfg.AccessSecret))
	if _, err := mac.Write([]byte(stringToSign)); err != nil {
		return errors.Wrap(err, "sign upload request")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, bytes.NewReader(content))
	if err != nil {
		return errors.Wrap(err, "create upload request")
	}

	req.Host = host
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Content-MD5", contentMD5)
	req.Header.Set("Date", date)
	req.Header.Set("Cache-Control", "public, max-age=31536000")
	req.Header.Set("Authorization", fmt.Sprintf(
		"OSS %s:%s",
		cfg.AccessID,
		base64.StdEncoding.EncodeToString(mac.Sum(nil)),
	))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "upload image to oss")
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	return errors.Errorf("upload image to oss failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
}

func (cfg bucketConfig) publicURL(objectKey string) string {
	publicHost := cfg.CDNHost
	if publicHost == "" {
		publicHost = cfg.endpointHost()
	}

	return (&url.URL{
		Scheme: cfg.PublicScheme,
		Host:   publicHost,
		Path:   "/" + objectKey,
	}).String()
}

func (cfg bucketConfig) objectURL(objectKey string) (string, string) {
	host := cfg.endpointHost()
	return (&url.URL{
		Scheme: cfg.UploadScheme,
		Host:   host,
		Path:   "/" + objectKey,
	}).String(), host
}

func (cfg bucketConfig) endpointHost() string {
	if strings.HasPrefix(cfg.Endpoint, cfg.Bucket+".") {
		return cfg.Endpoint
	}
	return cfg.Bucket + "." + cfg.Endpoint
}

func safePathSegment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "anonymous"
	}
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '-' || r == '_':
			return r
		default:
			return '-'
		}
	}, value)
}

func normalizeHost(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if strings.Contains(trimmed, "://") {
		if parsed, err := url.Parse(trimmed); err == nil && parsed.Host != "" {
			return parsed.Host
		}
	}
	return strings.Trim(trimmed, "/")
}

func endpointScheme(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if strings.HasPrefix(trimmed, "http://") {
		return "http"
	}
	return "https"
}
