package store

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
)

func GetProviderInfo(c *gin.Context) {
	providerInfoList, err := readProviderInfoList()
	if err != nil {

		fmt.Printf("failed while reading the provider info list, err: %v", err)
		c.JSON(404, gin.H{"message": fmt.Sprintf("failed to read provider list: %v", err)})
		return
	}

	c.JSON(200, providerInfoList)
}

func readProviderInfoList() ([]*ProviderInfo, error) {
	providerInfoListPath := "./depin/config.json"

	var providerInfoList []*ProviderInfo = make([]*ProviderInfo, 0)

	f, err := os.Open(providerInfoListPath)
	if err != nil {
		return nil, fmt.Errorf("unable to open file depin/config.json, err: %v", err)
	}
	defer f.Close()

	jsonDecoder := json.NewDecoder(f)
	if err := jsonDecoder.Decode(&providerInfoList); err != nil {
		return nil, err
	}
	
	return providerInfoList, nil
}