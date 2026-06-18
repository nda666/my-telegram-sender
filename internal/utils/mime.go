package utils

import (
	"path/filepath"
	"strings"
)

// mimeByExt returns MIME type dari ekstensi file.
// Fallback ke application/octet-stream kalau tidak dikenali.
func MimeByExt(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	m := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".webp": "image/webp",
		".gif":  "image/gif",
		".bmp":  "image/bmp",
		".mp4":  "video/mp4",
		".mov":  "video/quicktime",
		".avi":  "video/x-msvideo",
		".mkv":  "video/x-matroska",
		".webm": "video/webm",
		".mp3":  "audio/mpeg",
		".ogg":  "audio/ogg",
		".flac": "audio/flac",
		".wav":  "audio/wav",
		".aac":  "audio/aac",
		".opus": "audio/opus",
		".pdf":  "application/pdf",
		".zip":  "application/zip",
	}
	if mt, ok := m[ext]; ok {
		return mt
	}
	return "application/octet-stream"
}
