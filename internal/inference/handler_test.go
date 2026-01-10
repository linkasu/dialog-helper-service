package inference

import "testing"

func TestResolveContainerAudioType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		filename    string
		want        string
		wantErr     bool
	}{
		{
			name:        "webm by extension",
			contentType: "application/octet-stream",
			filename:    "recording.webm",
			want:        "WEBM_OPUS",
		},
		{
			name:        "webm by audio content type",
			contentType: "audio/webm;codecs=opus",
			filename:    "recording.bin",
			want:        "WEBM_OPUS",
		},
		{
			name:        "webm by video content type",
			contentType: "video/webm",
			filename:    "recording.bin",
			want:        "WEBM_OPUS",
		},
		{
			name:        "ogg",
			contentType: "audio/ogg",
			filename:    "sample.ogg",
			want:        "OGG_OPUS",
		},
		{
			name:        "mp3",
			contentType: "audio/mpeg",
			filename:    "sample.mp3",
			want:        "MP3",
		},
		{
			name:        "wav",
			contentType: "audio/wav",
			filename:    "sample.wav",
			want:        "WAV",
		},
		{
			name:        "unsupported type",
			contentType: "audio/flac",
			filename:    "sample.flac",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveContainerAudioType(tt.contentType, tt.filename)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
