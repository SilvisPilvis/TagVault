package database

import (
	"database/sql"
	"main/goexport/logger"

	_ "github.com/mattn/go-sqlite3"
)

// var Db *sql.DB = nil
var appLogger = logger.InitLogger()

func Init() *sql.DB {
	db, err := sql.Open("sqlite3", "file:./index.db?_timeout=10000&_busy_timeout=10000")
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
		"CREATE TABLE IF NOT EXISTS `Options`(`dbPath` VARCHAR(255) NOT NULL, `timezone` VARCHAR(1024) NOT NULL, `sortDesc` BOOLEAN);",
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

func GetImagesFromDatabase(db *sql.DB, imageCount uint) ([]string, error) {
	images, err := db.Query("SELECT path FROM Image ORDER BY dateAdded DESC LIMIT ?", imageCount)
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
	query := `
        SELECT DISTINCT Image.path
        FROM Image
        JOIN ImageTag ON Image.id = ImageTag.imageId
        JOIN Tag ON ImageTag.tagId = Tag.id
        WHERE Tag.name LIKE ?
    `

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
