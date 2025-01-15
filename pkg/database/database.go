package database

import (
	"context"
	"database/sql"
	"fmt"
	"main/pkg/fileutils"
	"main/pkg/imageconv"
	"main/pkg/logger"
	"main/pkg/options"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var appLogger = logger.InitLogger()
var currentTime = time.Now().Format("2006-01-02")
var userHome, _ = os.UserHomeDir()

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
		"CREATE TABLE IF NOT EXISTS `Tag`(`id` INTEGER PRIMARY KEY NOT NULL, `name` VARCHAR(255) NOT NULL UNIQUE, `color` VARCHAR(7) NOT NULL);",
		"CREATE TABLE IF NOT EXISTS `File`(`id` INTEGER PRIMARY KEY NOT NULL, `path` VARCHAR(1024) NOT NULL UNIQUE, `name` VARCHAR(256) NOT NULL UNIQUE, `md5` VARCHAR(32) NOT NULL UNIQUE, `dateAdded` DATETIME NOT NULL);",
		"CREATE INDEX IF NOT EXISTS idx_image_path ON File(path);", // Creates index on File.path to make searching by path faster
		"CREATE TABLE IF NOT EXISTS `FileTag`(`id` INTEGER PRIMARY KEY NOT NULL, `fileId` INTEGER NOT NULL, `tagId` INTEGER NOT NULL);",
		"CREATE TABLE IF NOT EXISTS `Options`(`id` INTEGER PRIMARY KEY NOT NULL, `DatabasePath` VARCHAR(255) NOT NULL, `ExcludedDirs` VARCHAR(255) NOT NULL, `Timezone` VARCHAR(1024) NOT NULL, `SortDesc` BOOLEAN DEFAULT true, `UseRGB` BOOLEAN DEFAULT false, `ImageNumber` INTEGER NOT NULL DEFAULT 20, `ThumbnailSize` INTEGER NOT NULL DEFAULT 256, `Profiling` BOOLEAN DEFAULT false, `ExifFields` VARCHAR(255), `FirstBoot` BOOLEAN DEFAULT false);",
		"PRAGMA journal_mode=WAL;",
	}
	for _, table := range tables {
		if _, err := db.Exec(table); err != nil {
			appLogger.Fatal("Failed to create table: ", err)
		}
	}
}

func VacuumDb(db *sql.DB) error {
	_, err := db.Exec("VACUUM")
	if err != nil {
		appLogger.Println("Failed to vacuum database: ", err)
		return err
	}

	return nil
}

func AddImageTypeTags(db *sql.DB) error {
	stmt, err := db.Prepare(`
    INSERT INTO Tag (name, color)
    SELECT ?, ?
    WHERE NOT EXISTS (
        SELECT 1 FROM Tag WHERE name = ?
    )
	`)
	if err != nil {
		return err
	}

	for i := 0; i < len(imageconv.ImageTypes); i++ {
		// Adds image type tags
		_, err = stmt.Exec(imageconv.ImageTypes[i], "#373c40", imageconv.ImageTypes[i])
		if err != nil {
			return err
		}
	}

	// Adds date added tag
	_, err = stmt.Exec(currentTime, "#373c40", currentTime)
	if err != nil {
		return err
	}

	return nil
}

func GetImageCount(db *sql.DB) int {
	var imgCount int
	count, err := db.Query("SELECT DISTINCT count(id) FROM File;")
	if err != nil {
		appLogger.Println("Error getting file count:", err)
	}
	count.Scan(&imgCount)
	return imgCount
}

func GetImagesFromDatabase(db *sql.DB, page int, imageCount uint) ([]string, error) {
	images, err := db.Query("SELECT path FROM File ORDER BY name DESC LIMIT ?,?", page, imageCount)
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
	err := db.QueryRow("SELECT id FROM File WHERE path = ?", path).Scan(&imageId)
	if err != nil {
		appLogger.Println("Error getting file ID:", err)
		return 0
	}
	return imageId
}

func GetDate(db *sql.DB, path string) string {
	var date string
	err := db.QueryRow("SELECT STRFTIME('%H:%M %d-%m-%Y', DATETIME(dateAdded, '+3 HOURS')) FROM File WHERE path = ?", path).Scan(&date)
	if err != nil {
		appLogger.Println("Error getting date:", err)
		return ""
	}
	return date
}

func GetImagePathsByTag(db *sql.DB, tagName string) ([]string, error) {
	query := `SELECT DISTINCT File.path FROM File JOIN FileTag ON File.id = FileTag.fileId JOIN Tag ON FileTag.tagId = Tag.id WHERE Tag.name LIKE ? OR File.name LIKE ?;`

	if tagName == "" || tagName == "%%" {
		return GetImagesFromDatabase(db, 0, 20)
	}

	stmt, err := db.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(tagName, tagName)
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
		SELECT DISTINCT File.path
		FROM File
		JOIN FileTag ON File.id = FileTag.fileId
		JOIN Tag ON FileTag.tagId = Tag.id
		WHERE Tag.name LIKE ?;
	`

	rows, err := db.Query(query, tagName)

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

func replaceHomeDir(path string) string {
	return strings.Replace(path, userHome, "~", 1)
}

func DiscoverImages(db *sql.DB, blacklist map[string]int) (bool, error) {
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
	stmt, err := db.PrepareContext(ctx, `
    INSERT INTO File (path, name, dateAdded, md5) 
    SELECT ?, ?, DATETIME('now'), ? 
    WHERE NOT EXISTS (SELECT 1 FROM File WHERE path = ?)
	`)
	if err != nil {
		return false, fmt.Errorf("error preparing SQL statement: %w", err)
	}
	defer stmt.Close()

	for _, directory := range directories {
		err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return fmt.Errorf("error walking path %s: %w", path, err)
			}
			if info.IsDir() && options.IsExcludedDir(path, blacklist) {
				var isExcluded int

				likePath := `"%` + path + `%"`

				err := db.QueryRow(`SELECT 1 FROM File WHERE path like ` + likePath + `;`).Scan(&isExcluded)
				if err != nil {
					if err == sql.ErrNoRows {
						appLogger.Println("Not in db.")
					}
				}

				if isExcluded == 1 {
					db.Exec(`DELETE FROM File WHERE path like ` + likePath + `;`)
				}

				// Skip path if path is a hidden dir or in excluded dirs
				appLogger.Println("Skipping hidden/blacklisted directory: ", replaceHomeDir(path))
				return filepath.SkipDir
			}
			if fileutils.IsImageFileMap(path) {
				// this needs to hash the whole image content not path
				imageHash, err := fileutils.GetFileMD5HashBuffered(path)
				if err != nil {
					return fmt.Errorf("error hashing image: %w", err)
				}

				// inserts image path into database
				insertId, err := stmt.Exec(path, strings.Split(filepath.Base(path), ".")[0], imageHash, path)
				if err != nil {
					return fmt.Errorf("failed to insert image into database: %w", err)
				}
				lastId, _ := insertId.LastInsertId()

				extension := filepath.Ext(path)[1:]
				extension = strings.ToUpper(extension)

				var extensionId int

				// check if extension is already in database
				db.QueryRow("SELECT id FROM Tag WHERE name = ?", extension).Scan(&extensionId)
				if extensionId != 0 {
					// add extension tag to image
					db.Exec(`INSERT INTO FileTag (fileId, tagId)
    				SELECT ?, ?
    				WHERE NOT EXISTS (
					SELECT 1 FROM FileTag
        			WHERE fileId = ? AND tagId = ?
    				)`, lastId, extensionId, lastId, extensionId)
				}

				// check if date tag in db
				var date int
				db.QueryRow("SELECT id from Tag where name like ?", currentTime+"%").Scan(&date)
				// var dateExists int
				// db.QueryRow("SELECT 1 FROM FileTag WHERE fileId = ? AND tagId = ?", lastId, date).Scan(&dateExists)
				// insert date in db if doesn't exist
				if date == 0 {
					dateInsert, _ := db.Exec("INSERT INTO Tag (name, color) VALUES (?, ?)", currentTime, "#373c40")
					dateId, _ := dateInsert.LastInsertId()
					if dateId != 0 {
						// insert date if date tag id is not 0
						db.Exec(`INSERT INTO FileTag (fileId, tagId)
						SELECT ?, ?
						WHERE NOT EXISTS (
						SELECT 1 FROM FileTag
						WHERE fileId = ? AND tagId = ?
						)`, lastId, dateId, lastId, dateId)
						// db.Exec("INSERT INTO FileTag (fileId, tagId) VALUES (?, ?)", lastId, dateId)
					}
				}
				// if date tag exists add date tag to image
				if date != 0 {
					db.Exec("INSERT INTO FileTag (fileId, tagId) VALUES (?, ?)", lastId, date)
				}

				count++
			}
			return nil
		})
		if err != nil {
			return false, fmt.Errorf("error walking directory %s: %w", directory, err)
		}
	}

	appLogger.Println("DISCOVERY COMPLETE. Added or Discovered ", count, " new files.")

	return true, nil
}

// Add a function to remove a tag from an image
func RemoveTagFromImage(db *sql.DB, imageId int, tagId int) error {
	_, err := db.Exec("DELETE FROM FileTag WHERE fileId = ? AND tagId = ?", imageId, tagId)
	return err
}

func GetTags(db *sql.DB) (map[int]string, error) {
	rows, err := db.Query("SELECT id, name FROM Tag")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tag map[int]string
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}

		if tag == nil {
			tag = make(map[int]string)
		}
		tag[id] = name
	}
	return tag, nil
}

func GetTagColorById(db *sql.DB, tagId int) (string, error) {
	var tagColor string
	err := db.QueryRow("SELECT color FROM Tag WHERE id = ?", tagId).Scan(&tagColor)
	return tagColor, err
}
