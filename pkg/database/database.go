package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
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
		"CREATE TABLE IF NOT EXISTS `Tag`(`id` INTEGER PRIMARY KEY NOT NULL, `name` VARCHAR(255) NOT NULL UNIQUE, `color` VARCHAR(7) NOT NULL);",
		"CREATE TABLE IF NOT EXISTS `File`(`id` INTEGER PRIMARY KEY NOT NULL, `path` VARCHAR(1024) NOT NULL UNIQUE, `md5` VARCHAR(32) NOT NULL UNIQUE, `dateAdded` DATETIME NOT NULL);",
		"CREATE INDEX IF NOT EXISTS idx_image_path ON File(path);", // Creates index on File.path to make searching by path faster
		"CREATE TABLE IF NOT EXISTS `FileTag`(`id` INTEGER PRIMARY KEY NOT NULL, `fileId` INTEGER NOT NULL, `tagId` INTEGER NOT NULL);",
		"CREATE TABLE IF NOT EXISTS `Options`(`id` INTEGER PRIMARY KEY NOT NULL, `DatabasePath` VARCHAR(255) NOT NULL, `ExcludedDirs` VARCHAR(255) NOT NULL, `Timezone` VARCHAR(1024) NOT NULL, `SortDesc` BOOLEAN DEFAULT true, `UseRGB` BOOLEAN DEFAULT false, `ImageNumber` INTEGER NOT NULL DEFAULT 20, `ThumbnailSize` INTEGER NOT NULL DEFAULT 256, `Profiling` BOOLEAN DEFAULT false, `ExifFields` VARCHAR(255), `FirstBoot` BOOLEAN DEFAULT false);",
		"PRAGMA journal_mode=WAL;",
		// "INSERT INTO `Tag` (`name`, `color`) VALUES ('GIF', '#000000'), ('JPG', '#000000'), ('PNG', '#000000'), ('AVIF', '#000000'), ('WEBP', '#000000'), ('BMP', '#000000'), ('HEIC', '#000000'), ('TIFF', '#000000'), ('TIF', '#000000'), ('QOI', '#000000');",
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
		_, err = stmt.Exec(imageconv.ImageTypes[i], "#373c40", imageconv.ImageTypes[i])
		if err != nil {
			return err
		}
	}

	// _, err = stmt.Exec("GIF", "#373c40", "GIF")
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
	images, err := db.Query("SELECT path FROM File ORDER BY dateAdded DESC LIMIT ?,?", page, imageCount)
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
	query := `SELECT DISTINCT File.path FROM File JOIN FileTag ON File.id = FileTag.fileId JOIN Tag ON FileTag.tagId = Tag.id WHERE Tag.name LIKE ?`

	// appLogger.Println("Searchable Tag: ", tagName)

	if tagName == "" || tagName == "%%" {
		// query = `
		// 	SELECT DISTINCT File.path
		// 	FROM File
		// 	JOIN FileTag ON File.id = FileTag.fileId
		// 	JOIN Tag ON FileTag.tagId = Tag.id
		// `
		return GetImagesFromDatabase(db, 0, 20)
	}

	stmt, err := db.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(tagName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// if tagName == "" || tagName == "%%" {
	// 	rows, err = stmt.Query(query)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	defer rows.Close()
	// }

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

func replaceHomeDir(path string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Println("Error getting user home directory:", err)
		return path
	}
	return strings.Replace(path, homeDir, "~", 1)
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
	stmt, err := db.PrepareContext(ctx, `
    INSERT INTO File (path, dateAdded, md5) 
    SELECT ?, DATETIME('now'), ? 
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
				// appLogger.Println("Is in db: ", isExcluded)

				if isExcluded == 1 {
					db.Exec(`DELETE FROM File WHERE path like ` + likePath + `;`)
				}

				// appLogger.Println("Skipping hidden/blacklisted directory: ", info.Name())
				appLogger.Println("Skipping hidden/blacklisted directory: ", replaceHomeDir(path))
				// Skip path if path is a hidden dir or in excluded dirs
				return filepath.SkipDir
			}
			if fileutils.IsImageFileMap(path) {
				// this needs to hash the whole image content not path
				imageHash, err := fileutils.GetFileMD5HashBuffered(path)
				if err != nil {
					return fmt.Errorf("error hashing image: %w", err)
				}

				// inserts image path into database
				insertId, err := stmt.Exec(path, imageHash, path)
				if err != nil {
					// appLogger.Println("Failed to insert image into database: ", err)
					return fmt.Errorf("failed to insert image into database: %w", err)
				}
				lastId, _ := insertId.LastInsertId()

				extension := filepath.Ext(path)[1:]
				// extension = strings.TrimPrefix(extension, ".") // replace with slice 1 from front instead
				extension = strings.ToUpper(extension)

				// appLogger.Println("Image Hash Len: ", len(imageHash))
				var extensionId int

				db.QueryRow("SELECT id FROM Tag WHERE name = ?", extension).Scan(&extensionId)

				db.Exec("INSERT INTO FileTag (fileId, tagId) VALUES (?, ?)", lastId, extensionId, imageHash)

				count++
			}
			// appLogger.Println("File not image or already in db:\n", path)
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
