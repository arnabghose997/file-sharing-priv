package store

import (
	"encoding/json"
	"fmt"
	"os"
)

type ProviderInfo struct {
	Storage     string `json:"storage"`
	Memory      string `json:"memory"`
	OS          string `json:"os"`
	Core        string `json:"core"`
	Processor   string `json:"processor"`
	ProviderDid string `json:"providerDid"`
}

func StoreDepinProviderInfo(provider *ProviderInfo) error {
	jsonFilePath := "./provider_info.json"

	providerList, err := readProviderInfoList()
	if err != nil {
		return fmt.Errorf("unable to read provider info list, err: %v", err)
	}

	providerList = append(providerList, provider)
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
