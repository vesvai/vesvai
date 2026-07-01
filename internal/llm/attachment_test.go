package llm

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestNewAttachmentFromBase64(t *testing.T) {
	att := NewAttachmentFromBase64(AttachmentTypeImage, "image/png", "base64data")
	if att.Type != AttachmentTypeImage {
		t.Errorf("Type = %q, want %q", att.Type, AttachmentTypeImage)
	}
	if att.MediaType != "image/png" {
		t.Errorf("MediaType = %q, want %q", att.MediaType, "image/png")
	}
	if att.Data != "base64data" {
		t.Errorf("Data = %q, want %q", att.Data, "base64data")
	}
	if att.URL != "" {
		t.Errorf("URL should be empty, got %q", att.URL)
	}
}

func TestNewAttachmentFromURL(t *testing.T) {
	att := NewAttachmentFromURL(AttachmentTypeAudio, "audio/mp3", "https://example.com/audio.mp3")
	if att.Type != AttachmentTypeAudio {
		t.Errorf("Type = %q, want %q", att.Type, AttachmentTypeAudio)
	}
	if att.MediaType != "audio/mp3" {
		t.Errorf("MediaType = %q, want %q", att.MediaType, "audio/mp3")
	}
	if att.URL != "https://example.com/audio.mp3" {
		t.Errorf("URL = %q, want %q", att.URL, "https://example.com/audio.mp3")
	}
	if att.Data != "" {
		t.Errorf("Data should be empty, got %q", att.Data)
	}
}

func TestNewImageAttachmentFromBase64(t *testing.T) {
	att := NewImageAttachmentFromBase64("image/jpeg", "rawdata")
	if att.Type != AttachmentTypeImage {
		t.Errorf("Type = %q, want %q", att.Type, AttachmentTypeImage)
	}
	if att.MediaType != "image/jpeg" {
		t.Errorf("MediaType = %q, want %q", att.MediaType, "image/jpeg")
	}
	if att.Data != "rawdata" {
		t.Errorf("Data = %q, want %q", att.Data, "rawdata")
	}
}

func TestNewImageAttachmentFromURL(t *testing.T) {
	att := NewImageAttachmentFromURL("https://example.com/photo.jpg")
	if att.Type != AttachmentTypeImage {
		t.Errorf("Type = %q, want %q", att.Type, AttachmentTypeImage)
	}
	if att.URL != "https://example.com/photo.jpg" {
		t.Errorf("URL = %q, want %q", att.URL, "https://example.com/photo.jpg")
	}
	// MediaType is empty for URL-based image
	if att.MediaType != "" {
		t.Errorf("MediaType = %q, want empty", att.MediaType)
	}
}

func TestNewAudioAttachmentFromBase64(t *testing.T) {
	att := NewAudioAttachmentFromBase64("audio/wav", "audiodata")
	if att.Type != AttachmentTypeAudio {
		t.Errorf("Type = %q, want %q", att.Type, AttachmentTypeAudio)
	}
	if att.MediaType != "audio/wav" {
		t.Errorf("MediaType = %q, want %q", att.MediaType, "audio/wav")
	}
	if att.Data != "audiodata" {
		t.Errorf("Data = %q, want %q", att.Data, "audiodata")
	}
}

func TestNewAudioAttachmentFromURL(t *testing.T) {
	att := NewAudioAttachmentFromURL("https://example.com/podcast.mp3")
	if att.Type != AttachmentTypeAudio {
		t.Errorf("Type = %q, want %q", att.Type, AttachmentTypeAudio)
	}
	if att.URL != "https://example.com/podcast.mp3" {
		t.Errorf("URL = %q, want %q", att.URL, "https://example.com/podcast.mp3")
	}
}

func TestNewFileAttachmentFromBase64(t *testing.T) {
	att := NewFileAttachmentFromBase64("application/pdf", "pdfdata", "doc.pdf")
	if att.Type != AttachmentTypeFile {
		t.Errorf("Type = %q, want %q", att.Type, AttachmentTypeFile)
	}
	if att.MediaType != "application/pdf" {
		t.Errorf("MediaType = %q, want %q", att.MediaType, "application/pdf")
	}
	if att.Data != "pdfdata" {
		t.Errorf("Data = %q, want %q", att.Data, "pdfdata")
	}
	if att.FileName != "doc.pdf" {
		t.Errorf("FileName = %q, want %q", att.FileName, "doc.pdf")
	}
}

func TestNewFileAttachmentFromURL(t *testing.T) {
	att := NewFileAttachmentFromURL("https://example.com/report.pdf", "report.pdf")
	if att.Type != AttachmentTypeFile {
		t.Errorf("Type = %q, want %q", att.Type, AttachmentTypeFile)
	}
	if att.URL != "https://example.com/report.pdf" {
		t.Errorf("URL = %q, want %q", att.URL, "https://example.com/report.pdf")
	}
	if att.FileName != "report.pdf" {
		t.Errorf("FileName = %q, want %q", att.FileName, "report.pdf")
	}
}

func TestEncodeFileToBase64(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{"empty", []byte{}, ""},
		{"hello", []byte("hello"), base64.StdEncoding.EncodeToString([]byte("hello"))},
		{"binary", []byte{0x00, 0xFF, 0x42}, base64.StdEncoding.EncodeToString([]byte{0x00, 0xFF, 0x42})},
		{"unicode", []byte("こんにちは"), base64.StdEncoding.EncodeToString([]byte("こんにちは"))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EncodeFileToBase64(tt.input)
			if got != tt.expected {
				t.Errorf("EncodeFileToBase64(%v) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestEncodeFileToBase64_RoundTrip(t *testing.T) {
	original := []byte("test data for round trip encoding")
	encoded := EncodeFileToBase64(original)
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("DecodeString error: %v", err)
	}
	if string(decoded) != string(original) {
		t.Errorf("round trip failed: got %q, want %q", decoded, original)
	}
}

func TestAttachmentTypeValues(t *testing.T) {
	tests := []struct {
		attType  AttachmentType
		expected string
	}{
		{AttachmentTypeImage, "image"},
		{AttachmentTypeAudio, "audio"},
		{AttachmentTypeFile, "file"},
	}
	for _, tt := range tests {
		if string(tt.attType) != tt.expected {
			t.Errorf("AttachmentType %v = %q, want %q", tt.attType, string(tt.attType), tt.expected)
		}
	}
}

func TestAttachment_JSONSerialization(t *testing.T) {
	att := Attachment{
		Type:      AttachmentTypeImage,
		MediaType: "image/png",
		Data:      "base64data",
		URL:       "https://example.com/img.png",
		FileName:  "photo.png",
	}
	data, err := json.Marshal(att)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}

	var decoded Attachment
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	if decoded.Type != AttachmentTypeImage {
		t.Errorf("Type = %q, want %q", decoded.Type, AttachmentTypeImage)
	}
	if decoded.MediaType != "image/png" {
		t.Errorf("MediaType = %q, want %q", decoded.MediaType, "image/png")
	}
	if decoded.Data != "base64data" {
		t.Errorf("Data = %q, want %q", decoded.Data, "base64data")
	}
}

func TestAttachment_JSONOmitEmpty(t *testing.T) {
	att := Attachment{
		Type: AttachmentTypeImage,
	}

	data, err := json.Marshal(att)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	// type is always present (not omitempty)
	if _, exists := decoded["type"]; !exists {
		t.Error("type field should be present")
	}
	// These should be omitted
	for _, key := range []string{"media_type", "data", "url", "file_name"} {
		if _, exists := decoded[key]; exists {
			t.Errorf("field %q should be omitted when empty", key)
		}
	}
}

func TestAttachment_AllTypesHaveCorrectJSONTag(t *testing.T) {
	att := Attachment{
		Type:      AttachmentTypeFile,
		MediaType: "text/plain",
		Data:      "SGVsbG8=",
		URL:       "https://example.com/doc.txt",
		FileName:  "doc.txt",
	}

	data, _ := json.Marshal(att)
	var raw map[string]string
	json.Unmarshal(data, &raw)

	if raw["type"] != "file" {
		t.Errorf("json type = %q, want %q", raw["type"], "file")
	}
	if raw["media_type"] != "text/plain" {
		t.Errorf("json media_type = %q, want %q", raw["media_type"], "text/plain")
	}
	if raw["data"] != "SGVsbG8=" {
		t.Errorf("json data = %q, want %q", raw["data"], "SGVsbG8=")
	}
	if raw["url"] != "https://example.com/doc.txt" {
		t.Errorf("json url = %q, want %q", raw["url"], "https://example.com/doc.txt")
	}
	if raw["file_name"] != "doc.txt" {
		t.Errorf("json file_name = %q, want %q", raw["file_name"], "doc.txt")
	}
}
