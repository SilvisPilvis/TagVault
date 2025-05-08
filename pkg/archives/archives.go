package archives

import (
	"archive/tar"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"

	// "errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"github.com/alexmullins/zip"
	"github.com/dsnet/compress/bzip2"
)

var ArchivePassword string

func CreateTarBzip2Archive(archivePath string, fileList []string, w fyne.Window) error {
	if len(fileList) <= 1 {
		// dialog.ShowError(errors.New("no files to archive"), w)
		return fmt.Errorf("no files to archive")
	}

	archive, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create archive: %w", err)
	}
	defer archive.Close()

	bzipWriter, err := bzip2.NewWriter(archive, &bzip2.WriterConfig{
		Level: bzip2.BestCompression,
	})
	if err != nil {
		dialog.ShowError(err, nil)
	}
	defer bzipWriter.Close() // Ensure bzipWriter is closed

	tarWriter := tar.NewWriter(bzipWriter)
	defer tarWriter.Close() // Ensure tarWriter is closed

	for _, filePath := range fileList {
		err := addFileToTarArchive(filePath, tarWriter)
		if err != nil {
			return fmt.Errorf("failed to add file %s to archive: %w", filePath, err)
		}
	}

	// Ensure tarWriter is closed properly
	if err := tarWriter.Close(); err != nil {
		return fmt.Errorf("failed to close tar writer: %w", err)
	}

	// Ensure gzipWriter/bzip2Writer is closed properly
	if err := bzipWriter.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	// Ensure archive is closed properly
	if err := archive.Close(); err != nil {
		return fmt.Errorf("failed to close archive: %w", err)
	}

	// Verify the archive is not empty
	info, err := os.Stat(archivePath)
	if err != nil {
		return fmt.Errorf("failed to stat archive file: %w", err)
	}

	fmt.Printf("Archive created successfully at %s with size %d bytes\n", archivePath, info.Size())
	return nil
}

func CreateTarGzipArchive(archivePath string, fileList []string, w fyne.Window) error {
	// Check if file list is empty
	if len(fileList) <= 1 {
		// dialog.ShowError(errors.New("no files to archive"), w)
		// if fileList is empty, return an error
		return fmt.Errorf("no files to archive")
	}

	// Create the archive file
	archive, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create archive: %w", err)
	}
	defer archive.Close()

	// Create a gzip writer for the archive
	gzipWriter := gzip.NewWriter(archive)
	defer gzipWriter.Close() // Ensure gzipWriter is closed to finalize the archive

	// Create a tar writer for the archive
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close() // Ensure tarWriter is closed

	// Loop through each file in selected files
	for _, filePath := range fileList {
		// Try to add the file to the archive
		err := addFileToTarArchive(filePath, tarWriter)
		if err != nil {
			//  If an error occurs, return it
			return fmt.Errorf("failed to add file %s to archive: %w", filePath, err)
		}
	}

	// Ensure tarWriter is closed properly
	if err := tarWriter.Close(); err != nil {
		return fmt.Errorf("failed to close tar writer: %w", err)
	}

	// Ensure gzipWriter/bzip2Writer is closed properly
	if err := gzipWriter.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	// Ensure archive is closed properly
	if err := archive.Close(); err != nil {
		return fmt.Errorf("failed to close archive: %w", err)
	}

	// Verify the archive is not empty
	info, err := os.Stat(archivePath)
	if err != nil {
		return fmt.Errorf("failed to stat archive file: %w", err)
	}

	// Debug message
	fmt.Printf("Archive created successfully at %s with size %d bytes\n", archivePath, info.Size())
	return nil
}

func CreateZipArchive(archivePath string, fileList []string, w fyne.Window) error {
	if len(fileList) <= 1 {
		// dialog.ShowError(errors.New("no files to archive"), w)
		return fmt.Errorf("no files to archive")
	}

	archive, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create archive: %w", err)
	}
	defer archive.Close()

	zipWriter := zip.NewWriter(archive)
	defer zipWriter.Close() // Ensure gzipWriter is closed to finalize the archive

	for _, filePath := range fileList {
		err := addFileZipToArchive(filePath, zipWriter)
		if err != nil {
			return fmt.Errorf("failed to add file %s to archive: %w", filePath, err)
		}
	}

	// Ensure zipWriter is closed properly
	if err := zipWriter.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	// Ensure archive is closed properly
	if err := archive.Close(); err != nil {
		return fmt.Errorf("failed to close archive: %w", err)
	}

	// Verify the archive is not empty
	info, err := os.Stat(archivePath)
	if err != nil {
		return fmt.Errorf("failed to stat archive file: %w", err)
	}

	fmt.Printf("Archive created successfully at %s with size %d bytes\n", archivePath, info.Size())
	return nil
}

func CreateEncryptedZipArchive(archivePath string, fileList []string, w fyne.Window) error {
	if len(fileList) <= 1 {
		// dialog.ShowError(errors.New("no files to archive"), w)
		return fmt.Errorf("no files to archive")
	}

	archive, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create archive: %w", err)
	}
	defer archive.Close()

	zipWriter := zip.NewWriter(archive)
	defer zipWriter.Close() // Ensure gzipWriter is closed to finalize the archive

	for _, filePath := range fileList {
		err := addEncryptedFileZipToArchive(filePath, zipWriter)
		if err != nil {
			return fmt.Errorf("failed to add file %s to archive: %w", filePath, err)
		}
	}

	// Ensure zipWriter is closed properly
	if err := zipWriter.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	// Ensure archive is closed properly
	if err := archive.Close(); err != nil {
		return fmt.Errorf("failed to close archive: %w", err)
	}

	// Verify the archive is not empty
	info, err := os.Stat(archivePath)
	if err != nil {
		return fmt.Errorf("failed to stat archive file: %w", err)
	}

	fmt.Printf("Archive created successfully at %s with size %d bytes\n", archivePath, info.Size())
	return nil
}

func CreateEncryptedTarBzip2Archive(archivePath string, fileList []string, w fyne.Window) error {
	if len(fileList) <= 1 {
		// dialog.ShowError(errors.New("no files to archive"), w)
		return fmt.Errorf("no files to archive")
	}

	archive, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create archive: %w", err)
	}
	defer archive.Close()

	bzipWriter, err := bzip2.NewWriter(archive, &bzip2.WriterConfig{
		Level: bzip2.BestCompression,
	})
	if err != nil {
		dialog.ShowError(err, nil)
	}
	defer bzipWriter.Close() // Ensure bzipWriter is closed

	tarWriter := tar.NewWriter(bzipWriter)
	defer tarWriter.Close() // Ensure tarWriter is closed

	for _, filePath := range fileList {
		err := addEncryptedFileToTarArchive(filePath, tarWriter)
		if err != nil {
			return fmt.Errorf("failed to add file %s to archive: %w", filePath, err)
		}
	}

	// Ensure tarWriter is closed properly
	if err := tarWriter.Close(); err != nil {
		return fmt.Errorf("failed to close tar writer: %w", err)
	}

	// Ensure gzipWriter/bzip2Writer is closed properly
	if err := bzipWriter.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	// Ensure archive is closed properly
	if err := archive.Close(); err != nil {
		return fmt.Errorf("failed to close archive: %w", err)
	}

	// Verify the archive is not empty
	info, err := os.Stat(archivePath)
	if err != nil {
		return fmt.Errorf("failed to stat archive file: %w", err)
	}

	fmt.Printf("Archive created successfully at %s with size %d bytes\n", archivePath, info.Size())
	return nil
}

func CreateEncryptedTarGzipArchive(archivePath string, fileList []string, w fyne.Window) error {
	if len(fileList) <= 1 {
		// dialog.ShowError(errors.New("no files to archive"), w)
		return fmt.Errorf("no files to archive")
	}

	archive, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create archive: %w", err)
	}
	defer archive.Close()

	gzipWriter := gzip.NewWriter(archive)
	defer gzipWriter.Close() // Ensure gzipWriter is closed to finalize the archive

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close() // Ensure tarWriter is closed

	for _, filePath := range fileList {
		err := addEncryptedFileToTarArchive(filePath, tarWriter)
		if err != nil {
			return fmt.Errorf("failed to add file %s to archive: %w", filePath, err)
		}
	}

	// Ensure tarWriter is closed properly
	if err := tarWriter.Close(); err != nil {
		return fmt.Errorf("failed to close tar writer: %w", err)
	}

	// Ensure gzipWriter/bzip2Writer is closed properly
	if err := gzipWriter.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	// Ensure archive is closed properly
	if err := archive.Close(); err != nil {
		return fmt.Errorf("failed to close archive: %w", err)
	}

	// Verify the archive is not empty
	info, err := os.Stat(archivePath)
	if err != nil {
		return fmt.Errorf("failed to stat archive file: %w", err)
	}

	fmt.Printf("Archive created successfully at %s with size %d bytes\n", archivePath, info.Size())
	return nil
}

func addFileToTarArchive(filePath string, tarWriter *tar.Writer) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info for %s: %w", filePath, err)
	}

	header, err := tar.FileInfoHeader(info, info.Name())
	if err != nil {
		return fmt.Errorf("failed to create tar header for %s: %w", filePath, err)
	}

	header.Name = filepath.Base(filePath)

	if err := tarWriter.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write tar header for %s: %w", filePath, err)
	}

	_, err = io.Copy(tarWriter, file) // you can replace _ with bytes and uncoment the print below to see info
	if err != nil {
		return fmt.Errorf("failed to write file content for %s: %w", filePath, err)
	}

	// fmt.Printf("Added %s to archive (size: %d bytes)\n", filePath, bytesWritten)
	return nil
}

func addFileZipToArchive(filePath string, zipWriter *zip.Writer) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info for %s: %w", filePath, err)
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return fmt.Errorf("failed to create tar header for %s: %w", filePath, err)
	}

	header.Name = filepath.Base(filePath)

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("failed to write zip header for %s: %w", filePath, err)
	}

	_, err = io.Copy(writer, file) // you can replace _ with bytes and uncoment the print below to see info
	if err != nil {
		return fmt.Errorf("failed to write file content for %s: %w", filePath, err)
	}

	// fmt.Printf("Added %s to archive (size: %d bytes)\n", filePath, bytesWritten)
	return nil
}

func addEncryptedFileZipToArchive(filePath string, zipWriter *zip.Writer) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info for %s: %w", filePath, err)
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return fmt.Errorf("failed to create tar header for %s: %w", filePath, err)
	}
	header.SetPassword(ArchivePassword)

	header.Name = filepath.Base(filePath)

	// --- Normal Zip Section Start
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("failed to write zip header for %s: %w", filePath, err)
	}

	_, err = io.Copy(writer, file) // you can replace _ with bytes and uncoment the print below to see info
	if err != nil {
		return fmt.Errorf("failed to write file content for %s: %w", filePath, err)
	}

	// fmt.Printf("Added %s to archive (size: %d bytes)\n", filePath, bytesWritten)
	return nil
	// --- Normal Zip Section End

	// hasher := sha256.New()
	// hasher.Write([]byte(archivePassword))
	// hashedPass := sha256.Sum256([]byte(archivePassword))

	// block, err := aes.NewCipher(hashedPass[:])
	// if err != nil {
	// 	return fmt.Errorf("failed to create cipher: %w", err)
	// }

	// gcm, err := cipher.NewGCM(block)
	// if err != nil {
	// 	return fmt.Errorf("failed to create GCM: %w", err)
	// }

	// nonce := make([]byte, gcm.NonceSize())
	// if _, err = rand.Read(nonce); err != nil {
	// 	return fmt.Errorf("failed to generate nonce: %w", err)
	// }

	// fileBytes, err := io.ReadAll(file)
	// if err != nil {
	// 	return fmt.Errorf("failed to read file content for %s: %w", filePath, err)
	// }

	// cipherText := gcm.Seal(nil, nonce, fileBytes, nil)

	// fullEncryptedContent := append(nonce, cipherText...)

	// encryptedWriter, err := zipWriter.CreateHeader(header)
	// if err != nil {
	// 	return fmt.Errorf("failed to write zip header for %s: %w", filePath, err)
	// }

	// _, err = encryptedWriter.Write(fullEncryptedContent)
	// if err != nil {
	// 	return fmt.Errorf("failed to write file content for %s: %w", filePath, err)
	// }

	// return nil
}

func addEncryptedFileToTarArchive(filePath string, tarWriter *tar.Writer) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info for %s: %w", filePath, err)
	}

	header, err := tar.FileInfoHeader(info, info.Name())
	if err != nil {
		return fmt.Errorf("failed to create tar header for %s: %w", filePath, err)
	}

	header.Name = filepath.Base(filePath)

	hasher := sha256.New()
	hasher.Write([]byte(ArchivePassword))
	hashedPass := sha256.Sum256([]byte(ArchivePassword))

	block, err := aes.NewCipher(hashedPass[:])
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read file content for %s: %w", filePath, err)
	}

	cipherText := gcm.Seal(nil, nonce, fileBytes, nil)

	fullEncryptedContent := append(nonce, cipherText...)

	// Update the size in the header to match the encrypted content
	header.Size = int64(len(fullEncryptedContent))

	if err := tarWriter.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write tar header for %s: %w", filePath, err)
	}

	_, err = tarWriter.Write(fullEncryptedContent)
	if err != nil {
		return fmt.Errorf("failed to write encrypted file content for %s: %w", filePath, err)
	}

	// fmt.Printf("Added %s to archive (size: %d bytes)\n", filePath, bytesWritten)

	return nil
}
