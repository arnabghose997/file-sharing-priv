package main

import (
	"bytes"
	"dapp/host/ft"
	"dapp/host/nft"
	"dapp/host/onboarding"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/gin-contrib/cache"
	"github.com/gin-contrib/cache/persistence"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	wasmbridge "github.com/rubixchain/rubix-wasm/go-wasm-bridge"
	wasmContext "github.com/rubixchain/rubix-wasm/go-wasm-bridge/context"
)

var TrieClientsMap = make(map[string]*websocket.Conn)

var Upgrader = websocket.Upgrader{
	// CheckOrigin allows connections from any origin, which is suitable for development
	// In production, this should be restricted to trusted origins
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization")
}

func main() {
	r := gin.Default()

	cacheStore := persistence.NewInMemoryStore(time.Second)

	r.GET("/ws", func(c *gin.Context) {
		handleSocketConnection(c.Writer, c.Request)
	})
	r.GET("/connected-clients", handleConnectedClients)
	r.GET("/ping-client", handlePingClient)

	r.POST("/api/upload_asset", handleUploadAsset)
	r.POST("/api/upload_asset/upload_artifacts", handleUploadAsset_UploadArtifacts)
	r.GET("/api/upload_asset/get_artifact_info_by_cid/:cid", cache.CachePage(cacheStore, 12*time.Hour, handleUploadAsset_GetArtifactInfo))
	r.GET("/api/upload_asset/get_artifact_file_name/:cid", cache.CachePage(cacheStore, 12*time.Hour, handleUploadAsset_GetArtifactFileName))

	r.POST("/api/use_asset", handleUseAsset)
	r.GET("/api/download_artifact/:cid", handleDownloadArtifact)

	r.POST("/api/pay_for_inference", handlePayForInference)

	r.POST("/api/onboard_infra_provider", handleUserOnboarding)
	r.GET("/api/onboarded_providers", handleOnboardedProviders)

	// NEW ENDPOINT FOR CREATE TOKEN
	r.POST("/api/create_token", handleCreateToken)

	// Metrics
	r.GET("/metrics/asset_count", handleMetricsAssetCount)
	r.GET("/metrics/transaction_count", cache.CachePage(cacheStore, 30*time.Second, handleMetricsTransactionCount))

	r.GET("/api/get_rating_by_asset", GetRatingsFromChain)

	r.Run(":8082")
}

func wrapError(f func(code int, obj any), msg string) {
	fmt.Println(msg)
	f(404, gin.H{"message": msg})
}

func wrapSuccess(f func(code int, obj any), msg string) {
	fmt.Println(msg)
	f(200, gin.H{"message": msg})
}

// wrapSuccessJSON sends a JSON response with status 200 and the provided object.
func wrapSuccessJSON(f func(code int, obj any), obj any) {
	f(200, obj)
}

type SmartContractDataReply struct {
	BlockNo            int    `json:"BlockNo"`
	BlockId            string `json:"BlockId"`
	SmartContractData  string `json:"SmartContractData"`
	Epoch              int64  `json:"Epoch"`
	InitiatorSignature string `json:"InitiatorSignature"`
	ExecutorDID        string `json:"ExecutorDID"`
	InitiatorSignData  string `json:"InitiatorSignData"`
}

type SmartContractDataRequest struct {
	Token  string `json:"token,omitempty"`
	Latest bool   `json:"latest"`
}

type SmartContractDataResponse struct {
	Status       bool                     `json:"status"`
	Message      string                   `json:"message"`
	Result       interface{}              `json:"result"`
	SCTDataReply []SmartContractDataReply `json:"SCTDataReply"`
}

type Rating struct {
	UserDID string `json:"user_did"`
	Rating  int    `json:"rating"`
	AssetID string `json:"asset_id"`
}

type WrappedRating struct {
	RateAsset Rating `json:"rate_asset"`
}

func RoundToPrecision(val float64, precision int, tolerance int) float64 {
	factor := math.Pow(10, float64(precision))
	return math.Round(val*factor) / factor
}

func GetRatingsFromChain(c *gin.Context) {
	w := http.ResponseWriter(c.Writer)
	enableCors(&w)

	assetID := c.Query("asset_id")
	if assetID == "" {
		wrapError(c.JSON, "asset_id is required")
		return
	}

	avg, userCount, err := GetRatingFromChain(assetID)
	if err != nil {
		fmt.Println(err)
		wrapSuccessJSON(c.JSON, map[string]interface{}{
			"average_rating": 0.0,
			"user_count":     userCount,
		})
		return
	}

	roundedAvg := RoundToPrecision(avg, 2, 1)

	wrapSuccessJSON(c.JSON, map[string]interface{}{
		"average_rating": roundedAvg,
		"user_count":     userCount,
	})

}

const RATING_CONTRACT_HASH = "QmfEkQvWcLZEghJ1swffQg9nxcnT13j6xLiB3CqPXUvfg2"

func GetRatingFromChain(assetID string) (float64, int, error) {
	reqBody := SmartContractDataRequest{
		Token:  RATING_CONTRACT_HASH,
		Latest: false,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return 0, 0, err
	}

	fmt.Printf("Sending request body to Rubix: %s\n", string(bodyBytes))

	resp, err := http.Post("http://localhost:20007/api/get-smart-contract-token-chain-data", "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, err
	}

	fmt.Printf("Raw API response: %s\n", string(respBody))

	var result SmartContractDataResponse
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		return 0, 0, err
	}

	if len(result.SCTDataReply) == 0 {
		return 0, 0, fmt.Errorf("Smart Contract Token %v is not registered", RATING_CONTRACT_HASH)
	}

	type userRating struct {
		Rating    int
		Epoch     int64
		UserCount int
	}

	latest := make(map[string]userRating)

	for _, reply := range result.SCTDataReply {
		if len(reply.SmartContractData) == 0 || reply.SmartContractData[0] != '{' {
			fmt.Printf("Skipping invalid SmartContractData: %s\n", reply.SmartContractData)
			continue
		}

		var wrapper WrappedRating
		err := json.Unmarshal([]byte(reply.SmartContractData), &wrapper)
		if err != nil {
			fmt.Printf("Failed to parse  SmartContractData: %s\n", reply.SmartContractData)
			continue
		}

		entry := wrapper.RateAsset
		if entry.AssetID == assetID && entry.Rating >= 1 && entry.Rating <= 5 {
			prev, exists := latest[entry.UserDID]
			if !exists || reply.Epoch > prev.Epoch {
				latest[entry.UserDID] = userRating{
					Rating: entry.Rating,
					Epoch:  reply.Epoch,
				}
			}
		}
	}

	if len(latest) == 0 {
		return 0, 0, errors.New("no valid ratings found for asset")
	}

	fmt.Println("Latest ratings per DID:")
	total := 0
	for did, ur := range latest {
		fmt.Printf("  %s -> %d (Epoch %d)\n", did, ur.Rating, ur.Epoch)
		total += ur.Rating
	}

	average := float64(total) / float64(len(latest))
	user_count := len(latest)
	return average, user_count, nil
}

func handleUploadAsset(c *gin.Context) {
	nodeAddress := "http://localhost:20007"
	quorumType := 2

	selfContractHashPath := path.Join("../artifacts/asset_publish_contract.wasm")

	var contractInputRequest ContractInputRequest

	err := json.NewDecoder(c.Request.Body).Decode(&contractInputRequest)
	if err != nil {
		wrapError(c.JSON, "err: Invalid request body")
		return
	}

	trieConn, ok := TrieClientsMap[contractInputRequest.InitiatorDID]
	if !ok {
		wrapError(c.JSON, fmt.Sprintf("clientID %s not found", contractInputRequest.InitiatorDID))
		return
	}

	wasmCtx := wasmContext.NewWasmContext().WithExternalSocketConn(trieConn)

	// Create Import function registry
	hostFnRegistry := wasmbridge.NewHostFunctionRegistry()
	hostFnRegistry.Register(ft.NewDoTransferFTApiCall())
	hostFnRegistry.Register(nft.NewDoMintNFTApiCall())
	hostFnRegistry.Register(ft.NewDoCreateFTApiCall())

	// Initialize the WASM module
	wasmModule, err := wasmbridge.NewWasmModule(
		selfContractHashPath,
		hostFnRegistry,
		wasmbridge.WithRubixNodeAddress(nodeAddress),
		wasmbridge.WithQuorumType(quorumType),
		wasmbridge.WithWasmContext(wasmCtx),
	)
	if err != nil {
		wrapError(c.JSON, fmt.Sprintf("unable to initialize wasmModule: %v", err))
		return
	}

	if contractInputRequest.SmartContractData == "" {
		wrapError(c.JSON, fmt.Sprintf("unable to fetch Smart Contract from callback"))
		return
	}

	_, err = wasmModule.CallFunction(contractInputRequest.SmartContractData)
	if err != nil {
		wrapError(c.JSON, fmt.Sprintf("unable to execute function, err: %v", err))
		return
	}
}

func handlePayForInference(c *gin.Context) {
	nodeAddress := "http://localhost:20007"
	quorumType := 2

	selfContractHashPath := path.Join("../artifacts/inference_contract.wasm")

	var contractInputRequest ContractInputRequest

	err := json.NewDecoder(c.Request.Body).Decode(&contractInputRequest)
	if err != nil {
		wrapError(c.JSON, "err: Invalid request body")
		return
	}

	trieConn, ok := TrieClientsMap[contractInputRequest.InitiatorDID]
	if !ok {
		wrapError(c.JSON, fmt.Sprintf("clientID %s not found", contractInputRequest.InitiatorDID))
		return
	}

	wasmCtx := wasmContext.NewWasmContext().WithExternalSocketConn(trieConn)

	// Create Import function registry
	hostFnRegistry := wasmbridge.NewHostFunctionRegistry()
	hostFnRegistry.Register(ft.NewDoTransferFTApiCall())

	// Initialize the WASM module
	wasmModule, err := wasmbridge.NewWasmModule(
		selfContractHashPath,
		hostFnRegistry,
		wasmbridge.WithRubixNodeAddress(nodeAddress),
		wasmbridge.WithQuorumType(quorumType),
		wasmbridge.WithWasmContext(wasmCtx),
	)
	if err != nil {
		wrapError(c.JSON, fmt.Sprintf("unable to initialize wasmModule: %v", err))
		return
	}

	if contractInputRequest.SmartContractData == "" {
		wrapError(c.JSON, fmt.Sprintf("unable to fetch Smart Contract from callback"))
		return
	}

	_, err = wasmModule.CallFunction(contractInputRequest.SmartContractData)
	if err != nil {
		wrapError(c.JSON, fmt.Sprintf("unable to execute function, err: %v", err))
		return
	}
}

func handleUseAsset(c *gin.Context) {
	nodeAddress := "http://localhost:20007"
	quorumType := 2

	selfContractHashPath := path.Join("../artifacts/asset_usage_contract.wasm")

	var contractInputRequest ContractInputRequest

	err := json.NewDecoder(c.Request.Body).Decode(&contractInputRequest)
	if err != nil {
		wrapError(c.JSON, "err: Invalid request body")
		return
	}

	trieConn, ok := TrieClientsMap[contractInputRequest.InitiatorDID]
	if !ok {
		wrapError(c.JSON, fmt.Sprintf("clientID %s not found", contractInputRequest.InitiatorDID))
		return
	}

	wasmCtx := wasmContext.NewWasmContext().WithExternalSocketConn(trieConn)

	// Create Import function registry
	hostFnRegistry := wasmbridge.NewHostFunctionRegistry()
	hostFnRegistry.Register(ft.NewDoTransferFTApiCall())
	hostFnRegistry.Register(nft.NewDoExecuteNFT())
	hostFnRegistry.Register(ft.NewDoCreateFTApiCall())

	// Initialize the WASM module
	wasmModule, err := wasmbridge.NewWasmModule(
		selfContractHashPath,
		hostFnRegistry,
		wasmbridge.WithRubixNodeAddress(nodeAddress),
		wasmbridge.WithQuorumType(quorumType),
		wasmbridge.WithWasmContext(wasmCtx),
	)
	if err != nil {
		wrapError(c.JSON, fmt.Sprintf("unable to initialize wasmModule: %v", err))
		return
	}

	if contractInputRequest.SmartContractData == "" {
		wrapError(c.JSON, fmt.Sprintf("unable to fetch Smart Contract from callback"))
		return
	}

	_, err = wasmModule.CallFunction(contractInputRequest.SmartContractData)
	if err != nil {
		wrapError(c.JSON, fmt.Sprintf("unable to execute function, err: %v", err))
		return
	}
}

// NEW HANDLER FOR CREATE TOKEN
func handleCreateToken(c *gin.Context) {
	nodeAddress := "http://localhost:20007"
	quorumType := 2

	// Use the existing WASM file that contains CREATE_FT functionality
	selfContractHashPath := path.Join("../artifacts/asset_create_ft.wasm")

	var contractInputRequest ContractInputRequest

	err := json.NewDecoder(c.Request.Body).Decode(&contractInputRequest)
	if err != nil {
		wrapError(c.JSON, "err: Invalid request body")
		return
	}
	fmt.Println("The value fo Initator DID: ", contractInputRequest.InitiatorDID)

	trieConn, ok := TrieClientsMap[contractInputRequest.InitiatorDID]
	if !ok {
		wrapError(c.JSON, fmt.Sprintf("clientID %s not found", contractInputRequest.InitiatorDID))
		return
	}

	wasmCtx := wasmContext.NewWasmContext().WithExternalSocketConn(trieConn)

	// Create Import function registry - only register what's needed for token creation
	hostFnRegistry := wasmbridge.NewHostFunctionRegistry()
	hostFnRegistry.Register(ft.NewDoCreateFTApiCall())

	// Initialize the WASM module
	wasmModule, err := wasmbridge.NewWasmModule(
		selfContractHashPath,
		hostFnRegistry,
		wasmbridge.WithRubixNodeAddress(nodeAddress),
		wasmbridge.WithQuorumType(quorumType),
		wasmbridge.WithWasmContext(wasmCtx),
	)
	if err != nil {
		wrapError(c.JSON, fmt.Sprintf("unable to initialize wasmModule: %v", err))
		return
	}

	if contractInputRequest.SmartContractData == "" {
		wrapError(c.JSON, fmt.Sprintf("unable to fetch Smart Contract from callback"))
		return
	}

	result, err := wasmModule.CallFunction(contractInputRequest.SmartContractData)
	if err != nil {
		wrapError(c.JSON, fmt.Sprintf("unable to execute function, err: %v", err))
		return
	}

	wrapSuccess(c.JSON, result)
}

func handleUserOnboarding(c *gin.Context) {
	nodeAddress := "http://localhost:20007"
	quorumType := 2

	selfContractHashPath := path.Join("../artifacts/onboarding_contract.wasm")

	var contractInputRequest ContractInputRequest

	err := json.NewDecoder(c.Request.Body).Decode(&contractInputRequest)
	if err != nil {
		wrapError(c.JSON, "err: Invalid request body")
		return
	}

	// Create Import function registry
	hostFnRegistry := wasmbridge.NewHostFunctionRegistry()
	hostFnRegistry.Register(onboarding.NewVerifyAction())
	// Initialize the WASM module
	wasmModule, err := wasmbridge.NewWasmModule(
		selfContractHashPath,
		hostFnRegistry,
		wasmbridge.WithRubixNodeAddress(nodeAddress),
		wasmbridge.WithQuorumType(quorumType),
	)
	if err != nil {
		wrapError(c.JSON, fmt.Sprintf("unable to initialize wasmModule: %v", err))
		return
	}

	if contractInputRequest.SmartContractData == "" {
		wrapError(c.JSON, fmt.Sprintf("unable to fetch Smart Contract from callback"))
		return
	}

	contractResult, err := wasmModule.CallFunction(contractInputRequest.SmartContractData)
	if err != nil {
		wrapError(c.JSON, fmt.Sprintf("unable to execute function, err: %v", err))
		return
	}

	msg, errMsg := extractSignatureVerificationOutput(contractResult)
	if errMsg != "" {
		wrapError(c.JSON, fmt.Sprintf("error occured while verifying the signature, err: %v", errMsg))
		return
	}

	switch msg {
	case "Success":
		wrapSuccess(c.JSON, "signature is valid")
		return
	case "Fail":
		wrapSuccess(c.JSON, "signature is invalid")
		return
	default:
		wrapError(c.JSON, fmt.Sprintf("unexpected error occured while retrieving the signature verification result, msg val extracted: %v", msg))
	}
}

func handleOnboardedProviders(c *gin.Context) {
	w := http.ResponseWriter(c.Writer)
	enableCors(&w)

	providerInfoPath := "./depin/config.json"

	providerInfo, err := os.ReadFile(providerInfoPath)
	if err != nil {
		getInternalError(c, fmt.Sprintf("failed to read depin/config.json file, err: %v", err))
		return
	}

	var providerInfoMap []map[string]interface{}
	err = json.Unmarshal(providerInfo, &providerInfoMap)
	if err != nil {
		getInternalError(c, fmt.Sprintf("failed to unmarshal depin/config.json file, err: %v", err))
		return
	}

	c.JSON(200, providerInfoMap)
}

func handleMetricsAssetCount(c *gin.Context) {
	w := http.ResponseWriter(c.Writer)
	enableCors(&w)

	targetURL, _ := url.JoinPath(RUBIX_API, "/api/list-nfts")

	response, err := queryRubixNode(targetURL)
	if err != nil {
		fmt.Printf("Failed to fetch NFT tokens: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"asset_count": 0, "ai_model_count": 0, "dataset_count": 0})
		return
	}

	var assetCountResponse AssetCountResponse

	if err := json.Unmarshal([]byte(response), &assetCountResponse); err != nil {
		fmt.Printf("Unable to unmarshal request for /api/list-nfts, err: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"asset_count": 0, "ai_model_count": 0, "dataset_count": 0})
		return
	}

	aiModelCount := 0
	datasetCount := 0

	rubixNftDir := os.Getenv("RUBIX_NFT_DIR")
	if rubixNftDir == "" {
		fmt.Printf("metricsAssetCount: Unable to get RUBIX_NFT_DIR env variable")
		c.JSON(http.StatusInternalServerError, gin.H{"asset_count": 0, "ai_model_count": 0, "dataset_count": 0})
		return
	}

	for _, nft := range assetCountResponse.Nfts {
		var intf struct {
			Type string `json:"type"`
		}

		if nft.NftMetadata != "" {
			if err := json.Unmarshal([]byte(nft.NftMetadata), &intf); err != nil {
				fmt.Printf("unable to unmarshal NFT metadata: %v", err)
				continue
			}
		} else {
			assetMetadataDir := path.Join(rubixNftDir, nft.Nft, "metadata.json")

			assetMetadataObj, err := os.ReadFile(assetMetadataDir)
			if err != nil {
				fmt.Printf("failed to read asset metadata file: %v", err)
				continue
			}

			if len(assetMetadataObj) == 0 {
				fmt.Printf("metadata file for NFT ID %v is empty", nft.Nft)
				continue
			}

			if err := json.Unmarshal(assetMetadataObj, &intf); err != nil {
				fmt.Printf("unable to unmarshal metadata JSON: %v", err)
				continue
			}
		}

		if intf.Type == "model" {
			aiModelCount++
		} else if intf.Type == "dataset" {
			datasetCount++
		} else {
			fmt.Printf("Unknown asset type for NFT ID %s: %s\n", nft.Nft, intf.Type)
		}
	}

	assetCount := aiModelCount + datasetCount
	c.JSON(http.StatusOK, gin.H{"asset_count": assetCount, "ai_model_count": aiModelCount, "dataset_count": datasetCount})
}

func handleMetricsTransactionCount(c *gin.Context) {
	w := http.ResponseWriter(c.Writer)
	enableCors(&w)

	supportedContracts := []string{
		"QmVRwuiYMES2vySvJwqZ1oFgxtDjWwQXWuhgTctgDNu9ye",
		"QmVAMKVR1Q9etqfwqfdSGWseNjRKdmHr6Zck2TL8MfeEyT",
		"QmfEkQvWcLZEghJ1swffQg9nxcnT13j6xLiB3CqPXUvfg2",
		"QmS5DogBfk96voS54hhE4KemToGRWgGC6Fbk5cZboTNh3m",
	}

	nfts, err := listNFTs()
	if err != nil {
		fmt.Printf("failed to fetch NFT tokens: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"transaction_count": 0})
		return
	}

	var totalTransactionCount int = 0

	for _, nftId := range nfts {
		transactions, err := listNFTTransactionsByID(nftId)
		if err != nil {
			fmt.Printf("failed to fetch NFT transactions for %s: %v\n", nftId, err)
			c.JSON(http.StatusInternalServerError, gin.H{"transaction_count": 0})
			return
		}

		if transactions.NFTDataReply == nil {
			continue
		}

		totalTransactionCount += len(transactions.NFTDataReply)
	}

	for _, contract := range supportedContracts {
		contractObj, err := listSmartContractTransactions(contract)
		if err != nil {
			fmt.Printf("contract %v, err: %v", contract, err)
			continue
		}

		if len(contractObj.SCTDataReply) == 0 {
			fmt.Println("Length of contract is zero: ", contract)
			continue
		}

		totalTransactionCount += len(contractObj.SCTDataReply)
	}

	// Send the JSON response
	c.JSON(http.StatusOK, gin.H{"transaction_count": totalTransactionCount})
}
