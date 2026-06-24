// files-module — Botspace module for common file operations.
package main

import (
	"fmt"

	botmodule "github.com/BotSpace/botmodule-go"
)

const moduleID = "files"

func main() {
	newModule().Serve(":8100")
}

func newModule() *botmodule.Module {
	m := botmodule.New(moduleID, "Files")
	m.Version = "0.1.0"
	m.Docs = docs

	m.AddNode(downloadURLNode())
	m.AddNode(deleteFileNode())
	m.AddNode(copyFileNode())
	m.AddNode(fileInfoNode())
	m.AddNode(readTextNode())

	return m
}

func downloadURLNode() botmodule.Node {
	return botmodule.Node{
		Type:        moduleID + ".DownloadURL",
		Title:       "Download file",
		Description: "Internet URL'dan fayl yuklab Botspace storage'ga saqlaydi",
		Category:    "integrations",
		Icon:        "download",
		Color:       "integration-sky",
		Content: []botmodule.Field{
			{Type: "text", Key: "url", Label: "File URL", Placeholder: "https://example.com/file.pdf"},
			{Type: "text", Key: "filename", Label: "Filename", Optional: true, Placeholder: "report.pdf"},
			{Type: "number", Key: "max_mb", Label: "Max MB", Optional: true, Placeholder: "25"},
			{Type: "number", Key: "timeout_seconds", Label: "Timeout", Optional: true, Placeholder: "30"},
			{Type: "number", Key: "ttl_seconds", Label: "Delete after seconds", Optional: true, Placeholder: "0", HelpText: "0 yoki bo'sh bo'lsa fayl doimiy saqlanadi."},
		},
		Defaults: map[string]any{
			"url":             "",
			"filename":        "",
			"max_mb":          25,
			"timeout_seconds": 30,
			"ttl_seconds":     0,
		},
		ProducesState: []string{"file_uuid", "file_url", "file_name", "file_size_bytes", "file_content_type", "file_source_url", "file_ttl_seconds", "file_error"},
		Outputs: []botmodule.Output{
			{Name: "success", Label: "Downloaded", Variant: "success"},
			{Name: "failed", Label: "Failed", Variant: "danger"},
		},
		Execute: func(c *botmodule.ExecuteCtx) botmodule.Result {
			file, err := DownloadRemoteFile(c.String("url"), c.String("filename"), c.Int("max_mb"), c.Int("timeout_seconds"))
			if err != nil {
				return failedResult(err)
			}

			ttlSeconds := normalizeTTL(c.Int("ttl_seconds"))
			uuid, err := c.UploadFileWithTTL(file.Name, file.Content, ttlSeconds)
			if err != nil {
				return failedResult(fmt.Errorf("upload file: %w", err))
			}

			return botmodule.Result{
				ExitOutput: "success",
				ContextUpdates: map[string]any{
					"file_uuid":         uuid,
					"file_url":          c.FileURL(uuid),
					"file_name":         file.Name,
					"file_size_bytes":   len(file.Content),
					"file_content_type": file.ContentType,
					"file_source_url":   file.SourceURL,
					"file_ttl_seconds":  ttlSeconds,
					"file_error":        "",
				},
			}
		},
	}
}

func deleteFileNode() botmodule.Node {
	return botmodule.Node{
		Type:        moduleID + ".DeleteFile",
		Title:       "Delete file",
		Description: "Botspace storage'dagi faylni UUID bo'yicha o'chiradi",
		Category:    "integrations",
		Icon:        "trash-2",
		Color:       "action-rose",
		Content: []botmodule.Field{
			{Type: "text", Key: "file_uuid", Label: "File UUID", Placeholder: "{{file_uuid}}"},
		},
		Defaults:      map[string]any{"file_uuid": "{{file_uuid}}"},
		ProducesState: []string{"deleted_file_uuid", "file_deleted", "file_error"},
		Outputs: []botmodule.Output{
			{Name: "deleted", Label: "Deleted", Variant: "success"},
			{Name: "failed", Label: "Failed", Variant: "danger"},
		},
		Execute: func(c *botmodule.ExecuteCtx) botmodule.Result {
			uuid := cleanUUID(c.String("file_uuid"))
			if uuid == "" {
				return failedResult(fmt.Errorf("file_uuid bo'sh"))
			}
			if err := c.DeleteFile(uuid); err != nil {
				return failedResult(err)
			}
			return botmodule.Result{
				ExitOutput: "deleted",
				ContextUpdates: map[string]any{
					"deleted_file_uuid": uuid,
					"file_deleted":      true,
					"file_error":        "",
				},
			}
		},
	}
}

func copyFileNode() botmodule.Node {
	return botmodule.Node{
		Type:        moduleID + ".CopyFile",
		Title:       "Copy file",
		Description: "Mavjud faylni o'qib, yangi UUID bilan qayta saqlaydi",
		Category:    "integrations",
		Icon:        "copy",
		Color:       "integration-green",
		Content: []botmodule.Field{
			{Type: "text", Key: "source_file_uuid", Label: "Source file UUID", Placeholder: "{{file_uuid}}"},
			{Type: "text", Key: "filename", Label: "New filename", Optional: true, Placeholder: "copy.bin"},
			{Type: "number", Key: "ttl_seconds", Label: "Delete after seconds", Optional: true, Placeholder: "0", HelpText: "0 yoki bo'sh bo'lsa yangi nusxa doimiy saqlanadi."},
		},
		Defaults:      map[string]any{"source_file_uuid": "{{file_uuid}}", "filename": "", "ttl_seconds": 0},
		ProducesState: []string{"file_uuid", "file_url", "source_file_uuid", "source_file_url", "file_name", "file_size_bytes", "file_content_type", "file_ttl_seconds", "file_error"},
		Outputs: []botmodule.Output{
			{Name: "success", Label: "Copied", Variant: "success"},
			{Name: "failed", Label: "Failed", Variant: "danger"},
		},
		Execute: func(c *botmodule.ExecuteCtx) botmodule.Result {
			sourceUUID := cleanUUID(c.String("source_file_uuid"))
			if sourceUUID == "" {
				return failedResult(fmt.Errorf("source_file_uuid bo'sh"))
			}

			content, err := c.GetFile(sourceUUID)
			if err != nil {
				return failedResult(err)
			}

			name := sanitizeFilename(c.String("filename"))
			if name == "" {
				name = "copy-" + sourceUUID + ".bin"
			}
			ttlSeconds := normalizeTTL(c.Int("ttl_seconds"))
			uuid, err := c.UploadFileWithTTL(name, content, ttlSeconds)
			if err != nil {
				return failedResult(fmt.Errorf("upload copy: %w", err))
			}

			return botmodule.Result{
				ExitOutput: "success",
				ContextUpdates: map[string]any{
					"file_uuid":         uuid,
					"file_url":          c.FileURL(uuid),
					"source_file_uuid":  sourceUUID,
					"source_file_url":   c.FileURL(sourceUUID),
					"file_name":         name,
					"file_size_bytes":   len(content),
					"file_content_type": detectContentType(content),
					"file_ttl_seconds":  ttlSeconds,
					"file_error":        "",
				},
			}
		},
	}
}

func fileInfoNode() botmodule.Node {
	return botmodule.Node{
		Type:        moduleID + ".FileInfo",
		Title:       "File info",
		Description: "Fayl UUID bo'yicha hajm va content type chiqaradi",
		Category:    "integrations",
		Icon:        "info",
		Color:       "integration-indigo",
		Content: []botmodule.Field{
			{Type: "text", Key: "file_uuid", Label: "File UUID", Placeholder: "{{file_uuid}}"},
		},
		Defaults:      map[string]any{"file_uuid": "{{file_uuid}}"},
		ProducesState: []string{"file_uuid", "file_url", "file_size_bytes", "file_content_type", "file_error"},
		Outputs: []botmodule.Output{
			{Name: "success", Label: "Found", Variant: "success"},
			{Name: "failed", Label: "Failed", Variant: "danger"},
		},
		Execute: func(c *botmodule.ExecuteCtx) botmodule.Result {
			uuid := cleanUUID(c.String("file_uuid"))
			if uuid == "" {
				return failedResult(fmt.Errorf("file_uuid bo'sh"))
			}
			content, err := c.GetFile(uuid)
			if err != nil {
				return failedResult(err)
			}
			return botmodule.Result{
				ExitOutput: "success",
				ContextUpdates: map[string]any{
					"file_uuid":         uuid,
					"file_url":          c.FileURL(uuid),
					"file_size_bytes":   len(content),
					"file_content_type": detectContentType(content),
					"file_error":        "",
				},
			}
		},
	}
}

func readTextNode() botmodule.Node {
	return botmodule.Node{
		Type:        moduleID + ".ReadText",
		Title:       "Read text file",
		Description: "Faylni text sifatida o'qib state'ga yozadi",
		Category:    "integrations",
		Icon:        "file-text",
		Color:       "action-emerald",
		Content: []botmodule.Field{
			{Type: "text", Key: "file_uuid", Label: "File UUID", Placeholder: "{{file_uuid}}"},
			{Type: "number", Key: "max_chars", Label: "Max chars", Optional: true, Placeholder: "4000"},
		},
		Defaults:      map[string]any{"file_uuid": "{{file_uuid}}", "max_chars": 4000},
		ProducesState: []string{"file_uuid", "file_url", "file_text", "file_text_truncated", "file_size_bytes", "file_content_type", "file_error"},
		Outputs: []botmodule.Output{
			{Name: "success", Label: "Read", Variant: "success"},
			{Name: "failed", Label: "Failed", Variant: "danger"},
		},
		Execute: func(c *botmodule.ExecuteCtx) botmodule.Result {
			uuid := cleanUUID(c.String("file_uuid"))
			if uuid == "" {
				return failedResult(fmt.Errorf("file_uuid bo'sh"))
			}
			content, err := c.GetFile(uuid)
			if err != nil {
				return failedResult(err)
			}
			text, truncated := textPreview(content, c.Int("max_chars"))
			return botmodule.Result{
				ExitOutput: "success",
				ContextUpdates: map[string]any{
					"file_uuid":           uuid,
					"file_url":            c.FileURL(uuid),
					"file_text":           text,
					"file_text_truncated": truncated,
					"file_size_bytes":     len(content),
					"file_content_type":   detectContentType(content),
					"file_error":          "",
				},
			}
		},
	}
}

func failedResult(err error) botmodule.Result {
	return botmodule.Result{
		ExitOutput: "failed",
		ContextUpdates: map[string]any{
			"file_error": err.Error(),
		},
	}
}

func normalizeTTL(ttlSeconds int64) int {
	if ttlSeconds <= 0 {
		return 0
	}
	const maxInt = int64(^uint(0) >> 1)
	if ttlSeconds > maxInt {
		return int(maxInt)
	}
	return int(ttlSeconds)
}

const docs = `# Files Module

Botspace file API bilan ishlash uchun Go moduli.

## Node turlari

### ` + "`files.DownloadURL`" + `

Internetdagi URL'dan fayl yuklab, Botspace storage'ga upload qiladi.

State:
- ` + "`file_uuid`" + `
- ` + "`file_url`" + `
- ` + "`file_name`" + `
- ` + "`file_size_bytes`" + `
- ` + "`file_content_type`" + `
- ` + "`file_source_url`" + `
- ` + "`file_ttl_seconds`" + `

` + "`ttl_seconds`" + ` > 0 bo'lsa fayl shuncha soniyadan keyin avtomatik o'chadi.
0 yoki bo'sh bo'lsa doimiy saqlanadi.

### ` + "`files.DeleteFile`" + `

` + "`file_uuid`" + ` bo'yicha faylni o'chiradi.

### ` + "`files.CopyFile`" + `

Mavjud faylni o'qib, yangi fayl sifatida qayta upload qiladi.

` + "`ttl_seconds`" + ` > 0 bo'lsa yangi nusxa shuncha soniyadan keyin avtomatik o'chadi.

State:
- ` + "`file_uuid`" + `
- ` + "`file_url`" + `
- ` + "`source_file_uuid`" + `
- ` + "`source_file_url`" + `
- ` + "`file_name`" + `
- ` + "`file_size_bytes`" + `
- ` + "`file_content_type`" + `
- ` + "`file_ttl_seconds`" + `

### ` + "`files.FileInfo`" + `

Faylni o'qib, public URL, hajm va MIME tipini aniqlaydi.

State:
- ` + "`file_uuid`" + `
- ` + "`file_url`" + `
- ` + "`file_size_bytes`" + `
- ` + "`file_content_type`" + `

### ` + "`files.ReadText`" + `

Fayl contentini text sifatida ` + "`file_text`" + ` state'ga yozadi.

State:
- ` + "`file_uuid`" + `
- ` + "`file_url`" + `
- ` + "`file_text`" + `

## Eslatma

Bu node'lar Botspace runtime tomonidan beriladigan file API bilan ishlaydi.
Lokal oddiy ` + "`go run .`" + ` paytida file API bo'lmasa upload/get/delete xato qaytaradi.
`
