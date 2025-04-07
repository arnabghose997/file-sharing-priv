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

	r.POST("/api/upload_asset", handleUploadAsset)
	r.GET("/ws", func(c *gin.Context) {
		handleSocketConnection(c.Writer, c.Request)
	})
	r.GET("/connected-clients", handleConnectedClients)
	r.GET("/ping-client", handlePingClient)

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
