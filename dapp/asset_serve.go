package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
)

func getResult(c *gin.Context, artifactPath string, metadataPath string) {
	c.JSON(http.StatusOK, gin.H{"status": true, "artifactPath": artifactPath, "metadataPath": metadataPath})
}

func getMetadataResult(c *gin.Context, artifactMetadata string) {
	c.JSON(http.StatusOK, gin.H{"status": true, "artifactMetadata": artifactMetadata})
}

func getArtifactResult(c *gin.Context, artifactFileName string) {
	c.JSON(http.StatusOK, gin.H{"status": true, "artifactFileName": artifactFileName})
}

func getInternalError(c *gin.Context, errMsg string) {
	c.JSON(http.StatusInternalServerError, gin.H{"status": false, "error": errMsg})
}

func getClientError(c *gin.Context, errMsg string) {
	c.JSON(http.StatusBadRequest, gin.H{"status": false, "error": errMsg})
}

// func handleUploadAsset_UploadArtifacts(c *gin.Context) {
// 	w := http.ResponseWriter(c.Writer)
// 	enableCors(&w)

// 	assetFile, err := c.FormFile("asset")
// 	if err != nil {
// 		getClientError(c, "Failed to get asset file, asset file is required")
// 		return
// 	}

// 	metadata, err := c.FormFile("metadata")
// 	if err != nil {
// 		getClientError(c, "Failed to get metadata file, metadata file is required")
// 		return
// 	}

// 	uploadDir := "./uploads"
// 	timeStampDir := fmt.Sprintf("%v", time.Now().Unix())
// 	uploadDestination := path.Join(uploadDir, timeStampDir)
// 	if err := os.MkdirAll(uploadDestination, 0755); err != nil {
// 		getInternalError(c, fmt.Sprintf("failed to create upload directory: %v", err))
// 		return
// 	}

// 	assetFilePath := path.Join(uploadDestination, assetFile.Filename)
// 	if err := c.SaveUploadedFile(assetFile, assetFilePath); err != nil {
// 		getInternalError(c, fmt.Sprintf("failed to save asset file: %v", err))
// 		return
// 	}

// 	metadataFilePath := path.Join(uploadDestination, metadata.Filename)
// 	if err := c.SaveUploadedFile(metadata, metadataFilePath); err != nil {
// 		getInternalError(c, fmt.Sprintf("failed to save metadata file: %v", err))
// 		return
// 	}

// 	getResult(c, assetFilePath, metadataFilePath)
// }

func (s *Server) handleUploadAsset_UploadArtifacts(c *gin.Context) {
	w := http.ResponseWriter(c.Writer)
	enableCors(&w)

	log.Printf("Request Content-Length: %d bytes", c.Request.ContentLength)

	// Get the multipart form
	form, err := c.MultipartForm()
	if err != nil {
		getClientError(c, "Failed to parse multipart form")
		return
	}
	defer form.RemoveAll() // Clean up temporary files

	// Get the asset file
	assetFiles, assetOk := form.File["asset"]
	if !assetOk || len(assetFiles) == 0 {
		getClientError(c, "Failed to get asset file, asset file is required")
		return
	}
	assetFile := assetFiles[0]

	// Get the metadata file
	metadataFiles, metadataOk := form.File["metadata"]
	if !metadataOk || len(metadataFiles) == 0 {
		getClientError(c, "Failed to get metadata file, metadata file is required")
		return
	}
	metadata := metadataFiles[0]

	// Create upload directory
	uploadDir := "./uploads"
	timeStampDir := fmt.Sprintf("%v", time.Now().Unix())
	uploadDestination := path.Join(uploadDir, timeStampDir)
	if err := os.MkdirAll(uploadDestination, 0755); err != nil {
		getInternalError(c, fmt.Sprintf("failed to create upload directory: %v", err))
		return
	}

	// Save asset file
	assetFilePath := path.Join(uploadDestination, assetFile.Filename)
	assetDest, err := os.Create(assetFilePath)
	if err != nil {
		getInternalError(c, fmt.Sprintf("failed to create asset file: %v", err))
		return
	}
	defer assetDest.Close()

	assetSrc, err := assetFile.Open()
	if err != nil {
		getInternalError(c, fmt.Sprintf("failed to open asset file: %v", err))
		return
	}
	defer assetSrc.Close()

	if _, err := io.Copy(assetDest, assetSrc); err != nil {
		getInternalError(c, fmt.Sprintf("failed to save asset file: %v", err))
		return
	}

	// Save metadata file
	metadataFilePath := path.Join(uploadDestination, metadata.Filename)
	metadataDest, err := os.Create(metadataFilePath)
	if err != nil {
		getInternalError(c, fmt.Sprintf("failed to create metadata file: %v", err))
		return
	}
	defer metadataDest.Close()

	metadataSrc, err := metadata.Open()
	if err != nil {
		getInternalError(c, fmt.Sprintf("failed to open metadata file: %v", err))
		return
	}
	defer metadataSrc.Close()

	if _, err := io.Copy(metadataDest, metadataSrc); err != nil {
		getInternalError(c, fmt.Sprintf("failed to save metadata file: %v", err))
		return
	}

	getResult(c, assetFilePath, metadataFilePath)
}

func (s *Server) handleUploadAsset_GetArtifactFileName(c *gin.Context) {
	w := http.ResponseWriter(c.Writer)
	enableCors(&w)

	assetCID := c.Param("cid")
	if assetCID == "" {
		getClientError(c, "cid is required, it came empty")
		return
	}

	rubixNftDir := os.Getenv("RUBIX_NFT_DIR")
	if rubixNftDir == "" {
		getInternalError(c, "RUBIX_NFT_DIR environment variable not set")
		return
	}

	assetMetadataDir := path.Join(rubixNftDir, assetCID)

	entries, err := os.ReadDir(assetMetadataDir)
	if err != nil {
		getInternalError(c, fmt.Sprintf("failed to read asset metadata file: %v", err))
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() && entry.Name() != "metadata.json" {
			getArtifactResult(c, entry.Name())
			return
		}
	}

	getInternalError(c, fmt.Sprintf("no artifact file found for NFT ID %v", assetCID))
}

func (s *Server) handleUploadAsset_GetArtifactInfo(c *gin.Context) {
	w := http.ResponseWriter(c.Writer)
	enableCors(&w)

	assetCID := c.Param("cid")
	if assetCID == "" {
		getClientError(c, "cid is required, it came empty")
		return
	}

	rubixNftDir := os.Getenv("RUBIX_NFT_DIR")
	if rubixNftDir == "" {
		getInternalError(c, "RUBIX_NFT_DIR environment variable not set")
		return
	}

	assetMetadataDir := path.Join(rubixNftDir, assetCID, "metadata.json")

	assetMetadataObj, err := os.ReadFile(assetMetadataDir)
	if err != nil {
		getInternalError(c, fmt.Sprintf("failed to read asset metadata file: %v", err))
		return
	}

	if len(assetMetadataObj) == 0 {
		getInternalError(c, fmt.Sprintf("metadata file for NFT ID %v is empty", assetCID))
		return
	}

	var intf map[string]interface{}

	if err := json.Unmarshal(assetMetadataObj, &intf); err != nil {
		getInternalError(c, fmt.Sprintf("unable to unmarshal metadata JSON: %v", err))
		return
	}

	assetMetadataJsonBytes, _ := json.Marshal(intf)

	base64EncodedMetadata := base64.StdEncoding.EncodeToString(assetMetadataJsonBytes)
	fmt.Println(base64EncodedMetadata)
	getMetadataResult(c, base64EncodedMetadata)
}

func (s *Server) handleDownloadArtifact(c *gin.Context) {
	w := http.ResponseWriter(c.Writer)
	enableCors(&w)

	assetCID := c.Param("cid")
	if assetCID == "" {
		getClientError(c, "cid is required, it came empty")
		return
	}

	rubixNftDir := os.Getenv("RUBIX_NFT_DIR")
	if rubixNftDir == "" {
		getInternalError(c, "RUBIX_NFT_DIR environment variable not set")
		return
	}

	assetDir := path.Join(rubixNftDir, assetCID)

	entries, err := os.ReadDir(assetDir)
	if err != nil {
		getInternalError(c, fmt.Sprintf("failed to read asset metadata file: %v", err))
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() && entry.Name() != "metadata.json" {
			c.File(path.Join(assetDir, entry.Name()))
			return
		}
	}

	getInternalError(c, fmt.Sprintf("no artifact file found for NFT ID %v", assetCID))
}
