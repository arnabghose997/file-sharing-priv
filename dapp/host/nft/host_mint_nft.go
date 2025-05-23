package nft

import (
	"encoding/json"
	"fmt"

	// "io/ioutil"

	"github.com/bytecodealliance/wasmtime-go"
	"github.com/gorilla/websocket"
	"github.com/rubixchain/rubix-wasm/go-wasm-bridge/context"
	"github.com/rubixchain/rubix-wasm/go-wasm-bridge/host"
	"github.com/rubixchain/rubix-wasm/go-wasm-bridge/utils"
)

type BasicResponse struct {
	Message string `json:"message"`
	Result  string `json:"result"`
	Status  bool   `json:"status"`
}

type DoMintNFTApiCall struct {
	allocFunc   *wasmtime.Func
	memory      *wasmtime.Memory
	nodeAddress string
	quorumType  int
	wasmContext *context.WasmContext
}

type MintNFTData struct {
	Did      string  `json:"did"`
	AssetId  string  `json:"assetId"`
	NftData  string  `json:"nftData"`
	NftValue float64 `json:"nftValue"`
}

type deployNFTReq struct {
	Nft        string  `json:"nft"`
	Did        string  `json:"did"`
	QuorumType int32   `json:"quorum_type"`
	NftData    string  `json:"nft_data"`
	NftValue   float64 `json:"nft_value"`
}

func NewDoMintNFTApiCall() *DoMintNFTApiCall {
	return &DoMintNFTApiCall{}
}

func (h *DoMintNFTApiCall) Name() string {
	return "do_mint_nft_trie"
}

func (h *DoMintNFTApiCall) FuncType() *wasmtime.FuncType {
	return wasmtime.NewFuncType(
		[]*wasmtime.ValType{
			wasmtime.NewValType(wasmtime.KindI32), // input_ptr
			wasmtime.NewValType(wasmtime.KindI32), // input_len
			wasmtime.NewValType(wasmtime.KindI32), // resp_ptr_ptr
			wasmtime.NewValType(wasmtime.KindI32), // resp_len_ptr
		},
		[]*wasmtime.ValType{wasmtime.NewValType(wasmtime.KindI32)}, // return i32
	)
}

func (h *DoMintNFTApiCall) Initialize(allocFunc, deallocFunc *wasmtime.Func, memory *wasmtime.Memory, nodeAddress string, quorumType int, wasmContext *context.WasmContext) {
	h.allocFunc = allocFunc
	h.memory = memory
	h.nodeAddress = nodeAddress
	h.quorumType = quorumType
	h.wasmContext = wasmContext
}

func (h *DoMintNFTApiCall) Callback() host.HostFunctionCallBack {
	return h.callback
}

func callDeployNFTAPI(webSocketConn *websocket.Conn, nodeAddress string, quorumType int, mintNFTData MintNFTData) (string, error) {
	var deployReq deployNFTReq

	deployReq.Did = mintNFTData.Did
	deployReq.Nft = mintNFTData.AssetId
	deployReq.QuorumType = int32(quorumType)
	deployReq.NftData = mintNFTData.NftData
	deployReq.NftValue = mintNFTData.NftValue

	deployNFTdataBytes, _ := json.Marshal(deployReq)
	var mintNFTDataMap map[string]interface{} = make(map[string]interface{})

	if err := json.Unmarshal(deployNFTdataBytes, &mintNFTDataMap); err != nil {
		return "", fmt.Errorf("error unmarshalling mintNFTBytes: %v", err)
	}

	msgPayload := map[string]interface{}{
		"type": "OPEN_EXTENSION",
		"data": &ExtensionCommand{
			Action:  "DEPLOY_NFT",
			Payload: mintNFTDataMap,
		},
	}

	msgPayloadBytes, _ := json.Marshal(msgPayload)

	// errDeadline := webSocketConn.SetReadDeadline(time.Now().Add(5 * time.Minute))
	// if errDeadline != nil {
	// 	return "", fmt.Errorf("error setting read deadline for web socket connection, err: %v", errDeadline)
	// }

	err := webSocketConn.WriteMessage(websocket.TextMessage, msgPayloadBytes)
	if err != nil {
		return "", fmt.Errorf("error occured while invoking Deploy NFT thrice, err: %v", err)
	}

	_, resultBytes, err := webSocketConn.ReadMessage()
	if err != nil {
		return "", fmt.Errorf("unable to read response from web socket connection for Deploy NFT, err: %v", err)
	}

	fmt.Println("Payload via websocket:", string(resultBytes))
	var basicResponse *BasicResponse
	if err := json.Unmarshal(resultBytes, &basicResponse); err != nil {
		return "", fmt.Errorf("unable to unmarshal the results for DeployNFT API call, err: %v", err)
	}

	txID, err := extractTransactionIDFromMessage(basicResponse.Message)
	if err != nil {
		return "", err
	}

	return txID, err
}

func (h *DoMintNFTApiCall) callback(
	caller *wasmtime.Caller,
	args []wasmtime.Val,
) ([]wasmtime.Val, *wasmtime.Trap) {
	trieServerSocketConnUrl := h.wasmContext.ExternalSocketConn()

	// Validate the number of arguments
	inputArgs, outputArgs := utils.HostFunctionParamExtraction(args, true, true)

	// Extract input bytes
	inputBytes, memory, err := utils.ExtractDataFromWASM(caller, inputArgs)
	if err != nil {
		fmt.Println("Failed to extract data from WASM", err)
		return utils.HandleError(err.Error())
	}
	h.memory = memory // Assign memory to Host struct for future use

	var mintNFTData MintNFTData
	//Unmarshaling the data which has been read from the wasm memory
	err3 := json.Unmarshal(inputBytes, &mintNFTData)
	if err3 != nil {
		fmt.Println("Error unmarshaling response in callback function:", err3)
		errMsg := "Error unmarshaling response in callback function:" + err3.Error()
		return utils.HandleError(errMsg)
	}

	nftDeployTxID, errDeploy := callDeployNFTAPI(trieServerSocketConnUrl, h.nodeAddress, h.quorumType, mintNFTData)
	if errDeploy != nil {
		errMsg := "Deploy NFT API failed" + errDeploy.Error()
		return utils.HandleError(errMsg)
	}

	responseStr := func() string {
		var data = struct {
			NftId string `json:"nftId"`
			TxId  string `json:"txId"`
		}{
			NftId: mintNFTData.AssetId,
			TxId:  nftDeployTxID,
		}

		dataBytes, _ := json.Marshal(data)
		return string(dataBytes)
	}()

	err = utils.UpdateDataToWASM(caller, h.allocFunc, responseStr, outputArgs)
	if err != nil {
		fmt.Println("Failed to update data to WASM", err)
		return utils.HandleError(err.Error())
	}

	return utils.HandleOk() // Success
}
