package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
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

func handleUploadAsset_UploadArtifacts(c *gin.Context) {
	w := http.ResponseWriter(c.Writer)
	enableCors(&w)

	assetFile, err := c.FormFile("asset")
	if err != nil {
		getClientError(c, "Failed to get asset file, asset file is required")
		return
	}

	metadata, err := c.FormFile("metadata")
	if err != nil {
		getClientError(c, "Failed to get metadata file, metadata file is required")
		return
	}

	uploadDir := "./uploads"
	timeStampDir := fmt.Sprintf("%v", time.Now().Unix())
	uploadDestination := path.Join(uploadDir, timeStampDir)
	if err := os.MkdirAll(uploadDestination, 0755); err != nil {
		getInternalError(c, fmt.Sprintf("failed to create upload directory: %v", err))
		return
	}

	assetFilePath := path.Join(uploadDestination, assetFile.Filename)
	if err := c.SaveUploadedFile(assetFile, assetFilePath); err != nil {
		getInternalError(c, fmt.Sprintf("failed to save asset file: %v", err))
		return
	}

	metadataFilePath := path.Join(uploadDestination, metadata.Filename)
	if err := c.SaveUploadedFile(metadata, metadataFilePath); err != nil {
		getInternalError(c, fmt.Sprintf("failed to save metadata file: %v", err))
		return
	}

	getResult(c, assetFilePath, metadataFilePath)
}

func handleUploadAsset_GetArtifactFileName(c *gin.Context) {
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

func handleUploadAsset_GetArtifactInfo(c *gin.Context) {
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
