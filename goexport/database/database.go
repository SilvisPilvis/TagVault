package database

import (
	"context"
	"database/sql"
	"fmt"
	"main/goexport/fileutils"
	"main/goexport/logger"
	"main/goexport/options"
	"os"
	"path/filepath"
	"runtime"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// var Db *sql.DB = nil
var appLogger = logger.InitLogger()

func Init() *sql.DB {
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s.db?timeout=10000&_busy_timeout=10000", runtime.GOOS))
	if err != nil {
		appLogger.Fatal("Failed to open database: ", err)
	}
	db.SetMaxOpenConns(2)
	if err := db.Ping(); err != nil {
		appLogger.Fatal("Failed to connect to database: ", err)
	}
	appLogger.Println("DB connection success!")

	setupTables(db)
	return db
}

func setupTables(db *sql.DB) {
	tables := []string{
		"CREATE TABLE IF NOT EXISTS `Tag`(`id` INTEGER PRIMARY KEY NOT NULL, `name` VARCHAR(255) NOT NULL, `color` VARCHAR(7) NOT NULL);",
		"CREATE TABLE IF NOT EXISTS `Image`(`id` INTEGER PRIMARY KEY NOT NULL, `path` VARCHAR(1024) NOT NULL, `dateAdded` DATETIME NOT NULL);",
		"CREATE INDEX IF NOT EXISTS idx_image_path ON Image(path);",
		"CREATE TABLE IF NOT EXISTS `ImageTag`(`imageId` INTEGER NOT NULL, `tagId` INTEGER NOT NULL);",
		"CREATE TABLE IF NOT EXISTS `Options`(`DatabasePath` VARCHAR(255) NOT NULL, `ExcludedDirs` VARCHAR(255) NOT NULL, `Timezone` VARCHAR(1024) NOT NULL, `SortDesc` BOOLEAN DEFAULT true, `UseRGB` BOOLEAN DEFAULT false, `ImageNumber` INTEGER NOT NULL DEFAULT 20, `ThumbnailSize` INTEGER NOT NULL DEFAULT 256, `Profiling` BOOLEAN DEFAULT false, `ExifFields` VARCHAR(255));",
	}
	for _, table := range tables {
		if _, err := db.Exec(table); err != nil {
			appLogger.Fatal("Failed to create table: ", err)
		}
	}
}

func GetImageCount(db *sql.DB) int {
	var imgCount int
	count, err := db.Query("SELECT DISTINCT count(id) FROM Image;")
	if err != nil {
		appLogger.Println("Error getting image count:", err)
	}
	count.Scan(&imgCount)
	return imgCount
}

func GetImagesFromDatabase(db *sql.DB, page int, imageCount uint) ([]string, error) {
	images, err := db.Query("SELECT path FROM Image ORDER BY dateAdded DESC LIMIT ?,?", page, imageCount)
	if err != nil {
		return nil, err
	}
	defer images.Close()

	var imagePaths []string
	for images.Next() {
		var path string
		if err := images.Scan(&path); err != nil {
			return nil, err
		}
		imagePaths = append(imagePaths, path)
	}

	return imagePaths, nil
}

func GetImageId(db *sql.DB, path string) int {
	var imageId int
	err := db.QueryRow("SELECT id FROM Image WHERE path = ?", path).Scan(&imageId)
	if err != nil {
		appLogger.Println("Error getting image ID:", err)
		return 0
	}
	return imageId
}

func GetDate(db *sql.DB, path string) string {
	var date string
	err := db.QueryRow("SELECT STRFTIME('%H:%M %d-%m-%Y', DATETIME(dateAdded, '+3 HOURS')) FROM Image WHERE path = ?", path).Scan(&date)
	if err != nil {
		appLogger.Println("Error getting date:", err)
		return ""
	}
	return date
}

func GetImagePathsByTag(db *sql.DB, tagName string) ([]string, error) {
	query := `SELECT DISTINCT Image.path FROM Image JOIN ImageTag ON Image.id = ImageTag.imageId JOIN Tag ON ImageTag.tagId = Tag.id WHERE Tag.name LIKE ?`
	// query := `
	// SELECT DISTINCT Image.path
	// FROM Image
	// JOIN ImageTag ON Image.id = ImageTag.imageId
	// JOIN Tag ON ImageTag.tagId = Tag.id
	// WHERE Tag.name LIKE ? OR Image.path LIKE ?
	// `

	stmt, err := db.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	// rows, err := stmt.Query(tagName, tagName)
	rows, err := stmt.Query(tagName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return paths, nil
}

// Function to handle tag-based search
func SearchImagesByTag(db *sql.DB, tagName string) ([]string, error) {
	query := `
		SELECT DISTINCT Image.path
		FROM Image
		JOIN ImageTag ON Image.id = ImageTag.imageId
		JOIN Tag ON ImageTag.tagId = Tag.id
		WHERE Tag.name LIKE ?
	`
	rows, err := db.Query(query, tagName)
	// rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var imagePaths []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		imagePaths = append(imagePaths, path)
	}

	return imagePaths, nil
}

func DiscoverImages(db *sql.DB, blacklist map[string]int) (bool, error) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return false, fmt.Errorf("error getting user home directory: %w", err)
	}

	var count int = 0

	appLogger.Println("Discovery started.")

	directories := []string{
		filepath.Join(userHome),
	}

	appLogger.Println("Home dir: ", directories)

	// adds context so we can cancel the operation
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	appLogger.Println("Created timeout context")
	stmt, err := db.PrepareContext(ctx, "INSERT INTO Image (path, dateAdded) SELECT ?, DATETIME('now') WHERE NOT EXISTS (SELECT 1 FROM Image WHERE path = ?)")
	if err != nil {
		return false, fmt.Errorf("error preparing SQL statement: %w", err)
	}
	defer stmt.Close()
	appLogger.Println("Prepared successfully")

	for _, directory := range directories {
		err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return fmt.Errorf("error walking path %s: %w", path, err)
			}
			if info.IsDir() && options.IsExcludedDir(path, blacklist) {
				appLogger.Println("Skipping hidden/exluded directory: ", info.Name())
				// Skip path if path is a hidden dir or in excluded dirs
				return filepath.SkipDir
			}
			if fileutils.IsImageFileMap(path) {
				_, err := stmt.Exec(path, path)
				if err != nil {
					return fmt.Errorf("error inserting image path into database: %w", err)
				}
				count++
			}
			appLogger.Println("File not image or already in db:\n", path)
			return nil
		})
		if err != nil {
			return false, fmt.Errorf("error walking directory %s: %w", directory, err)
		}
	}

	appLogger.Println("Discovery Complete. Added or Discovered ", count, " new images.")

	return true, nil
}

// Add a function to remove a tag from an image
func RemoveTagFromImage(db *sql.DB, imageId int, tagId int) error {
	_, err := db.Exec("DELETE FROM ImageTag WHERE imageId = ? AND tagId = ?", imageId, tagId)
	return err
}

// func init() {
// 	Db, err := sql.Open("sqlite3", "file:../index.db")
// 	if err != nil {
// 		panic(err.Error())
// 	}
// 	Db.SetMaxOpenConns(1)
// 	defer Db.Close()

// 	// check the connection
// 	err = Db.Ping()
// 	if err != nil {
// 		fmt.Print("Not Connected to db!\n")
// 		appLogger.Fatal(err.Error(), "\n")
// 	}
// 	fmt.Print("Connected to db!\n")
// 	// return Db
// }
