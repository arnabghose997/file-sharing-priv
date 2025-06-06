package store

import (
	"encoding/json"
	"fmt"
	"os"
)

type DepinEndpoints struct {
	Upload    string `json:"upload"`
	Inference string `json:"inference"`
}

type ProviderInfo struct {
	Storage             string         `json:"storage"`
	Memory              string         `json:"memory"`
	OS                  string         `json:"os"`
	Core                string         `json:"core"`
	GPU                 string         `json:"gpu"`
	Region              string         `json:"region"`
	PlatformName        string         `json:"platformName"`
	ProviderName        string         `json:"providerName"`
	PlatformDescription string         `json:"platformDescription"`
	PlatformImageUri    string         `json:"platformImageUri"`
	Processor           string         `json:"processor"`
	ProviderDid         string         `json:"providerDid"`
	HostingCost         int            `json:"hostingCost"`
	TrainingCost        int            `json:"trainingCost"`
	Endpoints           DepinEndpoints `json:"endpoints"`
	SupportedModels     string         `json:"supportedModels"`
}

func StoreDepinProviderInfo(provider *ProviderInfo) error {
	jsonFilePath := "./depin/config.json"

	providerList, err := readProviderInfoList()
	if err != nil {
		return fmt.Errorf("unable to read provider info list, err: %v", err)
	}

	// Check if an existing record for a provider DID exists or not
	// If yes, then edit the record
	existingRecord := false
	for idx, existingProvider := range providerList {
		if provider.ProviderDid == existingProvider.ProviderDid {
			providerList[idx] = provider
			existingRecord = true
		}
	}

	if !existingRecord {
		providerList = append(providerList, provider)
	}

	providerListBytes, err := json.MarshalIndent(providerList, "", " ")
	if err != nil {
		return fmt.Errorf("unable to marshal provider list, err: %v", err)
	}

	err = os.WriteFile(jsonFilePath, providerListBytes, 0644)
	if err != nil {
		return fmt.Errorf("unable to write file to data, err: %v", err)
	}

	return nil
}

func UnmarshalSmartContractData(smartContractDataStr string) (*ProviderInfo, error) {
	var smartContractDataMap map[string]map[string]interface{}

	err := json.Unmarshal([]byte(smartContractDataStr), &smartContractDataMap)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshall level1 Map")
	}

	fmt.Println(smartContractDataMap)

	providerInfoBytes, _ := json.Marshal(smartContractDataMap["onboard_provider"]["provider_info"])

	var providerInfo *ProviderInfo
	if err := json.Unmarshal(providerInfoBytes, &providerInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal provider info")
	}

	return providerInfo, nil
}
