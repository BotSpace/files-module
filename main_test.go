package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDownloadRemoteFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="hello.txt"`)
		_, _ = w.Write([]byte("hello"))
	}))
	defer server.Close()

	file, err := DownloadRemoteFile(server.URL+"/download", "", 1, 5)
	if err != nil {
		t.Fatalf("DownloadRemoteFile returned error: %v", err)
	}
	if file.Name != "hello.txt" {
		t.Fatalf("name = %q, want hello.txt", file.Name)
	}
	if string(file.Content) != "hello" {
		t.Fatalf("content = %q, want hello", string(file.Content))
	}
}

func TestReadLimitedRejectsLargeFile(t *testing.T) {
	_, err := readLimited(bytes.NewBufferString("abcdef"), 5)
	if err == nil {
		t.Fatal("expected size error")
	}
}

func TestTextPreview(t *testing.T) {
	text, truncated := textPreview([]byte("salom dunyo"), 5)
	if text != "salom" || !truncated {
		t.Fatalf("textPreview = %q, %v", text, truncated)
	}
}

func TestUploadFileKeepsPostAcrossSlashRedirect(t *testing.T) {
	var gotMethod string
	var gotBody string
	var gotTTL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/source" {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("hello"))
			return
		}
		if r.URL.Path == "/upload" {
			http.Redirect(w, r, "/upload/", http.StatusMovedPermanently)
			return
		}
		gotMethod = r.Method
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("ParseMultipartForm: %v", err)
		}
		gotTTL = r.FormValue("ttl")
		file, _, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("FormFile: %v", err)
		}
		defer file.Close()
		raw, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("ReadAll: %v", err)
		}
		gotBody = string(raw)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"uuid":"u123"}`))
	}))
	defer server.Close()

	rpc := httptest.NewServer(newModule().ServeHandler())
	defer rpc.Close()

	body := bytes.NewBufferString(`{
		"jsonrpc":"2.0",
		"method":"node.execute",
		"id":1,
		"params":{
			"type":"files.DownloadURL",
			"data":{"url":"` + server.URL + `/source","filename":"hello.txt","max_mb":1,"timeout_seconds":5,"ttl_seconds":60},
			"file_api":{"get_base":"` + server.URL + `/file","upload_url":"` + server.URL + `/upload","token":"token"}
		}
	}`)
	resp, err := http.Post(rpc.URL+"/rpc", "application/json", body)
	if err != nil {
		t.Fatalf("post node.execute: %v", err)
	}
	defer resp.Body.Close()

	var out struct {
		Result struct {
			ContextUpdates map[string]any `json:"context_updates"`
			ExitOutput     string         `json:"exit_output"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode execute: %v", err)
	}
	uuid, _ := out.Result.ContextUpdates["file_uuid"].(string)
	if uuid != "u123" {
		t.Fatalf("uuid = %q, want u123", uuid)
	}
	fileURL, _ := out.Result.ContextUpdates["file_url"].(string)
	if fileURL != server.URL+"/file/u123/" {
		t.Fatalf("file_url = %q, want %q", fileURL, server.URL+"/file/u123/")
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("redirected method = %q, want POST", gotMethod)
	}
	if gotBody != "hello" {
		t.Fatalf("body = %q, want hello", gotBody)
	}
	if gotTTL != "60" {
		t.Fatalf("ttl = %q, want 60", gotTTL)
	}
	if out.Result.ContextUpdates["file_ttl_seconds"] != float64(60) {
		t.Fatalf("file_ttl_seconds = %#v, want 60", out.Result.ContextUpdates["file_ttl_seconds"])
	}
}

func TestDescribeRPC(t *testing.T) {
	server := httptest.NewServer(newModule().ServeHandler())
	defer server.Close()

	body := bytes.NewBufferString(`{"jsonrpc":"2.0","method":"describe","id":1}`)
	resp, err := http.Post(server.URL+"/rpc", "application/json", body)
	if err != nil {
		t.Fatalf("post describe: %v", err)
	}
	defer resp.Body.Close()

	var out struct {
		Result struct {
			Module struct {
				ID string `json:"id"`
			} `json:"module"`
			Nodes []struct {
				Type string `json:"type"`
			} `json:"nodes"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode describe: %v", err)
	}
	if out.Result.Module.ID != moduleID {
		t.Fatalf("module ID = %q, want %q", out.Result.Module.ID, moduleID)
	}
	if len(out.Result.Nodes) != 5 {
		t.Fatalf("nodes length = %d, want 5", len(out.Result.Nodes))
	}
}
