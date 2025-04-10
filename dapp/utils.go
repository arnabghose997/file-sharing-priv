package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

const HOSTING_LIST_INFO = "./asset_holding_list_info.json"

type HostingFileInfo = map[string]bool

func GetHostingListInfo() (HostingFileInfo, error) {
	hostingFileInfo := make(HostingFileInfo)

	assetHostingListInfo, err := os.ReadFile(HOSTING_LIST_INFO)
	if err != nil {
		return nil, fmt.Errorf("failed to read asset_holding_list_info.json file, err: %v", err)
	}

	if err = json.Unmarshal(assetHostingListInfo, &hostingFileInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal asset_holding_list_info.json file, err: %v", err)
	}

	return hostingFileInfo, nil
}

func AddNFTRecordToHostingListInfo(nftId string) error {
	// Open the JSON file
	file, err := os.Open(HOSTING_LIST_INFO)
	if err != nil {
		if os.IsNotExist(err) {
			// If file doesn't exist, create a new one with the key
			initialData := HostingFileInfo{nftId: true}
			return writeJSON(initialData)
		}
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read file contents
	byteValue, err := ioutil.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Unmarshal into HostingFileInfo
	var data HostingFileInfo
	if len(byteValue) > 0 {
		err = json.Unmarshal(byteValue, &data)
		if err != nil {
			return fmt.Errorf("error unmarshaling JSON: %w", err)
		}
	} else {
		data = make(HostingFileInfo) // Initialize if file is empty
	}

	// Add the new key
	data[nftId] = true

	// Write updated data back to file
	return writeJSON(data)
}

func writeJSON(data HostingFileInfo) error {
	// Convert to JSON format
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %w", err)
	}

	// Write to file using os.WriteFile
	err = os.WriteFile(HOSTING_LIST_INFO, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write JSON file: %w", err)
	}

	return nil
}

type successMsg = string
type failMsg = string

func extractSignatureVerificationOutput(msg string) (successMsg, failMsg) {
	msgPrefix := "msg: "
	errPrefix := ", err: "

	msgStart := len(msgPrefix)
	msgEnd := strings.Index(msg, errPrefix)

	message := msg[msgStart:msgEnd]
	errorMsg := msg[msgEnd+len(errPrefix):]

	return strings.TrimSpace(message), strings.TrimSpace(errorMsg)
}
