package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const RUBIX_API = "http://localhost:20007"

type AssetCountResponse struct {
	BasicResponse
	Nfts []struct {
		Nft      string  `json:"nft"`
		NftValue float64 `json:"nft_value"`
		OwnerDID string  `json:"owner_did"`
		NftMetadata string `json:"nft_metadata"`
		NFTFileName string `json:"nft_file_name"`
	} `json:"nfts"`
}

type NFTTransactionList struct {
	BasicResponse
	NFTDataReply []interface{} `json:"NFTDataReply"`
}

type SCTransactionList struct {
	BasicResponse
	SCDataReply []interface{} `json:"SCDataReply"`
}

func fetchFromRubixNode(url, contentType string, data []byte) (string, error) {
	resp, err := http.Post(url, contentType, bytes.NewBuffer(data))
	if err != nil {
		return "", fmt.Errorf("POST request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(bodyBytes), nil
}

func queryRubixNode(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("GET request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(bodyBytes), nil
}

func listNFTs() ([]string, error) {
	targetURL, err := url.JoinPath(RUBIX_API, "/api/list-nfts")
	if err != nil {
		return nil, fmt.Errorf("failed to construct URL: %w", err)
	}

	response, err := queryRubixNode(targetURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch NFT tokens: %w", err)
	}

	var assetCountResponse AssetCountResponse
	if err := json.Unmarshal([]byte(response), &assetCountResponse); err != nil {
		return nil, fmt.Errorf("unable to unmarshal response: %w", err)
	}

	var nftList []string
	for _, nft := range assetCountResponse.Nfts {
		nftList = append(nftList, nft.Nft)
	}

	return nftList, nil
}

func listSmartContractTransactions(contractId string) (*SmartContractDataResponse, error) {
	targetURL, err := url.JoinPath(RUBIX_API, "/api/get-smart-contract-token-chain-data")
	if err != nil {
		return nil, fmt.Errorf("failed form POST request, err: %v", err)
	}

	requestParam := map[string]interface{}{
		"token":  contractId,
		"latest": false,
	}

	requestParamBytes, _ := json.Marshal(requestParam)

	response, err := fetchFromRubixNode(targetURL, "application/json", requestParamBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch NFT transactions: %w", err)
	}

	var transactions *SmartContractDataResponse
	if err := json.Unmarshal([]byte(response), &transactions); err != nil {
		return nil, fmt.Errorf("SC: unable to unmarshal response: %w", err)
	}

	return transactions, nil
}

func listNFTTransactionsByID(nftId string) (*NFTTransactionList, error) {
	baseURL, err := url.Parse(RUBIX_API)
	if err != nil {
		return nil, fmt.Errorf("failed to construct URL: %w", err)
	}
	baseURL.Path = "/api/get-nft-token-chain-data"
	query := baseURL.Query()
	query.Set("nft", nftId)
	baseURL.RawQuery = query.Encode()
	targetURL := baseURL.String()

	response, err := queryRubixNode(targetURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch NFT transactions: %w", err)
	}

	var transactions *NFTTransactionList
	if err := json.Unmarshal([]byte(response), &transactions); err != nil {
		return nil, fmt.Errorf("unable to unmarshal response: %w", err)
	}

	return transactions, nil
}
