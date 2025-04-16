package nft

import (
	"encoding/json"
	"fmt"

	"github.com/bytecodealliance/wasmtime-go"
	"github.com/rubixchain/rubix-wasm/go-wasm-bridge/context"
	"github.com/rubixchain/rubix-wasm/go-wasm-bridge/host"
	"github.com/rubixchain/rubix-wasm/go-wasm-bridge/utils"

	"github.com/gorilla/websocket"
	wasmContext "github.com/rubixchain/rubix-wasm/go-wasm-bridge/context"
)

type ExecuteNFTReq struct {
	NFT        string  `json:"nft"`
	Executor   string  `json:"executor"`
	Receiver   string  `json:"receiver"`
	Comment    string  `json:"comment"`
	NFTValue   float64 `json:"nft_value"`
	NFTData    string  `json:"nft_data"`
	QuorumType int32   `json:"quorum_type"`
}

type DoExecuteNFT struct {
	allocFunc   *wasmtime.Func
	memory      *wasmtime.Memory
	nodeAddress string
	quorumType  int
	wasmContext *context.WasmContext
}

func NewDoExecuteNFT() *DoExecuteNFT {
	return &DoExecuteNFT{}
}
func (h *DoExecuteNFT) Name() string {
	return "do_execute_nft"
}
func (h *DoExecuteNFT) FuncType() *wasmtime.FuncType {
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

func (h *DoExecuteNFT) Initialize(allocFunc, deallocFunc *wasmtime.Func, memory *wasmtime.Memory, nodeAddress string, quorumType int, wasmCtx *wasmContext.WasmContext) {
	h.allocFunc = allocFunc
	h.memory = memory
	h.nodeAddress = nodeAddress
	h.quorumType = quorumType
	h.wasmContext = wasmCtx
}

func (h *DoExecuteNFT) Callback() host.HostFunctionCallBack {
	return h.callback
}
func callExecuteNFTAPI(webSocketConn *websocket.Conn, nodeAddress string, quorumType int, executeNFTdata ExecuteNFTReq) error {
	executeNFTdata.QuorumType = int32(quorumType)
	fmt.Println("printing the data in callExecuteNFTAPI function is:", executeNFTdata)

	executeNFTdataBytes, _ := json.Marshal(executeNFTdata)
	var executeNFTDataMap map[string]interface{} = make(map[string]interface{})

	if err := json.Unmarshal(executeNFTdataBytes, &executeNFTDataMap); err != nil {
		return fmt.Errorf("error unmarshalling transferFTdataBytes: %v", err)
	}

	msgPayload := map[string]interface{}{
		"type": "OPEN_EXTENSION",
		"data": &ExtensionCommand{
			Action:  "EXECUTE_NFT",
			Payload: executeNFTDataMap,
		},
	}

	// errDeadline := webSocketConn.SetReadDeadline(time.Now().Add(5 * time.Minute))
	// if errDeadline != nil {
	// 	return fmt.Errorf("error setting read deadline for web socket connection, err: %v", errDeadline)
	// }

	msgPayloadBytes, _ := json.Marshal(msgPayload)

	err := webSocketConn.WriteMessage(websocket.TextMessage, msgPayloadBytes)
	if err != nil {
		return fmt.Errorf("error occured while invoking NFT Execute thrice, err: %v", err)
	}

	_, _, err = webSocketConn.ReadMessage()
	if err != nil {
		return fmt.Errorf("unable to read response from web socket connection for Execute NFT, err: %v", err)
	}

	return err
}

func (h *DoExecuteNFT) callback(
	caller *wasmtime.Caller,
	args []wasmtime.Val,
) ([]wasmtime.Val, *wasmtime.Trap) {
	trieSocketConn := h.wasmContext.ExternalSocketConn()
	if trieSocketConn == nil {
		return utils.HandleError("websocket connection for DoExecuteNFT is not initialized")
	}

	inputArgs, outputArgs := utils.HostFunctionParamExtraction(args, true, true)

	// Extract input bytes and convert to string
	inputBytes, memory, err := utils.ExtractDataFromWASM(caller, inputArgs)
	if err != nil {
		fmt.Println("Failed to extract data from WASM", err)
		return utils.HandleError(err.Error())
	}
	h.memory = memory // Assign memory to Host struct for future use
	var executeNFTData ExecuteNFTReq

	//Unmarshaling the data which has been read from the wasm memory
	err3 := json.Unmarshal(inputBytes, &executeNFTData)
	if err3 != nil {
		fmt.Println("Error unmarshaling response in callback function:", err3)
		errMsg := "Error unmashalling response in callback function" + err3.Error()
		return utils.HandleError(errMsg)
	}
	callExecuteNFTAPIRespErr := callExecuteNFTAPI(trieSocketConn, h.nodeAddress, h.quorumType, executeNFTData)
	if callExecuteNFTAPIRespErr != nil {
		fmt.Println("failed to execute NFT", callExecuteNFTAPIRespErr)
		errMsg := "failed to execute NFT" + callExecuteNFTAPIRespErr.Error()
		return utils.HandleError(errMsg)
	}

	responseStr := "success"
	err = utils.UpdateDataToWASM(caller, h.allocFunc, responseStr, outputArgs)
	if err != nil {
		fmt.Println("Failed to update data to WASM", err)
		return utils.HandleError(err.Error())
	}

	return utils.HandleOk() // Success

}
