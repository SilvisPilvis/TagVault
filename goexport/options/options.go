package options

import (
	"os"
	"path/filepath"
	"strings"
)

type Options struct {
	DatabasePath  string
	ExcludedDirs  map[string]int
	Profiling     bool
	Timezone      int // Timezone like UTC+3 or UTC-3
	SortDesc      bool
	UseRGB        bool
	ExifFields    []string // exif fields to display in the sidebar
	ImageNumber   uint
	ThumbnailSize int
}

func IsExcludedDir(dir string, blackList map[string]int) bool {
	// checks if the directory is blacklisted
	for key := range blackList {
		if strings.Contains(dir, key) {
			return true
		}
	}
	// checks if the path is a hidden directory
	return strings.HasPrefix(filepath.Base(dir), ".")
}

func (opts Options) InitDefault() *Options {
	return &Options{
		DatabasePath:  "./index.db",
		ExcludedDirs:  map[string]int{},
		Profiling:     false,
		Timezone:      3,
		SortDesc:      true,
		UseRGB:        false,
		ExifFields:    []string{"DateTime"},
		ImageNumber:   20,
		ThumbnailSize: 256,
	}
}

func (opts Options) SaveOptions() {
	os.OpenFile("application.options", os.O_RDWR|os.O_CREATE, 0666)
}

func (opts Options) LoadOptions() *Options {
	return &Options{
		DatabasePath: "./index.db",
		ExcludedDirs: map[string]int{},
		Profiling:    false,
		Timezone:     3,
		SortDesc:     true,
		UseRGB:       false,
		ExifFields:   []string{"DateTime"},
		ImageNumber:  20,
	}
}
