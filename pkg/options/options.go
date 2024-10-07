package options

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
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
	FirstBoot     bool
}

// Checks if the directory is blacklisted
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
		DatabasePath:  fmt.Sprintf("./%s.db", runtime.GOOS),
		ExcludedDirs:  map[string]int{},
		Profiling:     false,
		Timezone:      3,
		SortDesc:      true,
		UseRGB:        false,
		ExifFields:    []string{"DateTime"},
		ImageNumber:   20,
		ThumbnailSize: 256,
		FirstBoot:     true,
	}
}

func CheckOptionsExists(db *sql.DB) (bool, error) {
	// Execute SQL statement
	rows, err := db.Query("SELECT * FROM Options;")
	if err != nil {
		return false, fmt.Errorf("error executing statement: %v", err)
	}
	defer rows.Close()

	return rows.Next(), nil
}

func SaveOptionsToDB(db *sql.DB, options *Options) error {
	// Convert map and slice to JSON for storage
	excludedDirsJSON, err := json.Marshal(options.ExcludedDirs)
	if err != nil {
		return fmt.Errorf("error marshaling ExcludedDirs: %v", err)
	}

	exifFieldsJSON, err := json.Marshal(options.ExifFields)
	if err != nil {
		return fmt.Errorf("error marshaling ExifFields: %v", err)
	}

	var numOptionsDb int64
	err = db.QueryRow("SELECT COUNT(*) FROM Options").Scan(&numOptionsDb)
	if err != nil {
		return fmt.Errorf("error getting number of options: %v", err)
	}

	var query string
	switch numOptionsDb {
	case 0:
		options.FirstBoot = true
		query = `
		INSERT INTO Options (
			DatabasePath, ExcludedDirs, Profiling, Timezone, SortDesc, 
			UseRGB, ExifFields, ImageNumber, ThumbnailSize, FirstBoot
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`
	case 1:
		options.FirstBoot = false
		query = `
		UPDATE Options SET
		DatabasePath = ?,
		ExcludedDirs = ?,
		Profiling = ?,
		Timezone = ?,
		SortDesc = ?,
		UseRGB = ?,
		ExifFields = ?,
		ImageNumber = ?,
		ThumbnailSize = ?,
		FirstBoot = ?
		WHERE id = 1;
		`
	default:
		return fmt.Errorf("error getting number of options: %v", err)
	}

	// Prepare SQL statement
	stmt, err := db.Prepare(query)
	if err != nil {
		return fmt.Errorf("error preparing statement: %v", err)
	}
	defer stmt.Close()

	// Execute SQL statement
	_, err = stmt.Exec(
		options.DatabasePath,
		string(excludedDirsJSON),
		options.Profiling,
		options.Timezone,
		options.SortDesc,
		options.UseRGB,
		string(exifFieldsJSON),
		options.ImageNumber,
		options.ThumbnailSize,
		options.FirstBoot,
	)
	if err != nil {
		return fmt.Errorf("error executing statement: %v", err)
	}

	return nil
}

func LoadOptionsFromDB(db *sql.DB) (*Options, error) {
	options := &Options{}

	row := db.QueryRow(`
		SELECT DatabasePath, ExcludedDirs, Profiling, Timezone, SortDesc, 
			   UseRGB, ExifFields, ImageNumber, ThumbnailSize, FirstBoot
		FROM options WHERE id = 1 LIMIT 1
	`)

	var excludedDirsJSON, exifFieldsJSON string

	err := row.Scan(
		&options.DatabasePath,
		&excludedDirsJSON,
		&options.Profiling,
		&options.Timezone,
		&options.SortDesc,
		&options.UseRGB,
		&exifFieldsJSON,
		&options.ImageNumber,
		&options.ThumbnailSize,
		&options.FirstBoot,
	)
	options.FirstBoot = false
	if err != nil {
		if err == sql.ErrNoRows {
			return options, nil // Return default options if no row found
		}
		return nil, fmt.Errorf("error scanning row: %v", err)
	}

	// Unmarshal JSON data
	err = json.Unmarshal([]byte(excludedDirsJSON), &options.ExcludedDirs)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling ExcludedDirs: %v", err)
	}

	err = json.Unmarshal([]byte(exifFieldsJSON), &options.ExifFields)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling ExifFields: %v", err)
	}

	return options, nil
}
