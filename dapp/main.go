package main

import (
	"dapp/host/ft"
	"dapp/host/nft"
	"encoding/json"
	"fmt"
	"net/http"
	"path"

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

	r.GET("/ws", func(c *gin.Context) {
		handleSocketConnection(c.Writer, c.Request)
	})
	r.GET("/connected-clients", handleConnectedClients)
	r.GET("/ping-client", handlePingClient)

	r.POST("/api/upload_asset", handleUploadAsset)
	r.POST("/api/upload_asset/upload_artifacts", handleUploadAsset_UploadArtifacts)
	r.GET("/api/upload_asset/get_artifact_info_by_cid/:cid", handleUploadAsset_GetArtifactInfo)
	r.GET("/api/upload_asset/get_artifact_file_name/:cid", handleUploadAsset_GetArtifactFileName)

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

func handleUploadAsset(c *gin.Context) {
	nodeAddress := "http://localhost:20011"
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
	nodeAddress := "http://localhost:20009"
	quorumType := 2

	selfContractHashPath := path.Join("../artifact/asset_publish_contract.wasm")

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

	ipfsHash, err := wasmModule.CallFunction(contractInputRequest.SmartContractData)
	if err != nil {
		wrapError(c.JSON, fmt.Sprintf("unable to execute function, err: %v", err))
		return
	}

	// TODO: Send a websocket message to TRIE to handle the transaction
	fmt.Println(ipfsHash)
}
