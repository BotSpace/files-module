package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	defaultMaxMB          = 25
	defaultTimeoutSeconds = 30
	maxAllowedMB          = 100
)

var filenameUnsafePattern = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

type RemoteFile struct {
	Name        string
	ContentType string
	Content     []byte
	SourceURL   string
}

func DownloadRemoteFile(rawURL, preferredName string, maxMB, timeoutSeconds int64) (*RemoteFile, error) {
	if maxMB <= 0 {
		maxMB = defaultMaxMB
	}
	if maxMB > maxAllowedMB {
		maxMB = maxAllowedMB
	}
	if timeoutSeconds <= 0 {
		timeoutSeconds = defaultTimeoutSeconds
	}

	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("file URL noto'g'ri")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("faqat http yoki https URL qo'llanadi")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("request yaratilmadi: %w", err)
	}
	req.Header.Set("User-Agent", "BotspaceFilesModule/0.1")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("file yuklab bo'lmadi: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("file server %d status qaytardi", resp.StatusCode)
	}

	limit := maxMB * 1024 * 1024
	if resp.ContentLength > limit {
		return nil, fmt.Errorf("file juda katta: %d bytes > %d bytes", resp.ContentLength, limit)
	}

	content, err := readLimited(resp.Body, limit)
	if err != nil {
		return nil, err
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = detectContentType(content)
	}

	name := sanitizeFilename(preferredName)
	if name == "" {
		name = filenameFromResponse(resp, parsed, contentType)
	}

	return &RemoteFile{
		Name:        name,
		ContentType: contentType,
		Content:     content,
		SourceURL:   resp.Request.URL.String(),
	}, nil
}

func readLimited(r io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, fmt.Errorf("file o'qishda xato: %w", err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("file juda katta: limit %d bytes", limit)
	}
	return data, nil
}

func filenameFromResponse(resp *http.Response, parsed *url.URL, contentType string) string {
	if disposition := resp.Header.Get("Content-Disposition"); disposition != "" {
		_, params, err := mime.ParseMediaType(disposition)
		if err == nil {
			if name := sanitizeFilename(params["filename"]); name != "" {
				return name
			}
		}
	}

	name := sanitizeFilename(path.Base(parsed.Path))
	if name == "" || name == "." || name == "/" {
		name = "download"
	}
	if !strings.Contains(path.Base(name), ".") {
		if exts, _ := mime.ExtensionsByType(strings.Split(contentType, ";")[0]); len(exts) > 0 {
			name += exts[0]
		}
	}
	return name
}

func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	name = strings.Trim(name, ".")
	name = path.Base(name)
	name = filenameUnsafePattern.ReplaceAllString(name, "_")
	name = strings.Trim(name, "._-")
	if len(name) > 180 {
		ext := path.Ext(name)
		base := strings.TrimSuffix(name, ext)
		if len(ext) > 20 {
			ext = ""
		}
		if keep := 180 - len(ext); keep > 0 && len(base) > keep {
			base = base[:keep]
		}
		name = base + ext
	}
	return name
}

func cleanUUID(uuid string) string {
	return strings.TrimSpace(uuid)
}

func detectContentType(content []byte) string {
	if len(content) == 0 {
		return "application/octet-stream"
	}
	return http.DetectContentType(content)
}

func textPreview(content []byte, maxChars int64) (string, bool) {
	if maxChars <= 0 {
		maxChars = 4000
	}
	text := strings.ToValidUTF8(string(content), "")
	if !utf8.ValidString(text) {
		text = strings.ToValidUTF8(text, "")
	}

	runes := []rune(text)
	if int64(len(runes)) <= maxChars {
		return text, false
	}
	if maxChars < 0 {
		return "", true
	}
	return string(runes[:maxChars]), true
}

func validateFileContent(content []byte) error {
	if len(content) == 0 {
		return errors.New("file bo'sh")
	}
	return nil
}
