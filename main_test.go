package main

import (
	"bytes"
	"encoding/json"
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
