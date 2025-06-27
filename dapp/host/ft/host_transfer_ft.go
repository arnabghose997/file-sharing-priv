package ft

import (
	"encoding/json"
	"fmt"

	"github.com/bytecodealliance/wasmtime-go"
	"github.com/gorilla/websocket"
	"github.com/rubixchain/rubix-wasm/go-wasm-bridge/context"
	"github.com/rubixchain/rubix-wasm/go-wasm-bridge/host"
	"github.com/rubixchain/rubix-wasm/go-wasm-bridge/utils"
)

type TransferFTData struct {
	FTCount    int32  `json:"ft_count"`
	FTName     string `json:"ft_name"`
	CreatorDID string `json:"creatorDID"`
	QuorumType int32  `json:"quorum_type"`
	Comment    string `json:"comment"`
	Receiver   string `json:"receiver"`
	Sender     string `json:"sender"`
}

type DoTransferFTApiCall struct {
	allocFunc   *wasmtime.Func
	memory      *wasmtime.Memory
	nodeAddress string
	quorumType  int
	wasmCtx     *context.WasmContext
}

func NewDoTransferFTApiCall() *DoTransferFTApiCall {
	return &DoTransferFTApiCall{}
}
func (h *DoTransferFTApiCall) Name() string {
	return "do_transfer_ft_trie"
}
func (h *DoTransferFTApiCall) FuncType() *wasmtime.FuncType {
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

func (h *DoTransferFTApiCall) Initialize(allocFunc, deallocFunc *wasmtime.Func, memory *wasmtime.Memory, nodeAddress string, quorumType int, wasmContext *context.WasmContext) {
	h.allocFunc = allocFunc
	h.memory = memory
	h.nodeAddress = nodeAddress
	h.quorumType = quorumType
	h.wasmCtx = wasmContext
}

func (h *DoTransferFTApiCall) Callback() host.HostFunctionCallBack {
	return h.callback
}
func callTransferFTAPI(webSocketConn *websocket.Conn, nodeAddress string, quorumType int, transferFTdata TransferFTData) error {
	fmt.Println("LOG: call from contract to do Transfer FT")
	transferFTdata.QuorumType = int32(quorumType)

	transferFTdataBytes, _ := json.Marshal(transferFTdata)
	var transferFTDataMap map[string]interface{} = make(map[string]interface{})

	if err := json.Unmarshal(transferFTdataBytes, &transferFTDataMap); err != nil {
		return fmt.Errorf("error unmarshalling transferFTdataBytes: %v", err)
	}

	// errDeadline := webSocketConn.SetReadDeadline(time.Now().Add(5 * time.Minute))
	// if errDeadline != nil {
	// 	return fmt.Errorf("error setting read deadline for web socket connection, err: %v", errDeadline)
	// }

	msgPayload := map[string]interface{}{
		"type": "OPEN_EXTENSION",
		"data": &ExtensionCommand{
			Action:  "TRANSFER_FT",
			Payload: transferFTDataMap,
		},
	}

	msgPayloadBytes, _ := json.Marshal(msgPayload)

	err := webSocketConn.WriteMessage(websocket.TextMessage, msgPayloadBytes)
	if err != nil {
		return fmt.Errorf("error occured while invoking FT transfer twice, err: %v", err)
	}

	_, resp, err := webSocketConn.ReadMessage()
	if err != nil {
		return fmt.Errorf("unable to read response from web socket connection for FT Transfer, err: %v", err)
	}

	fmt.Println("Response received for FT Transfer:", string(resp))

	var response *BasicResponse
	err3 := json.Unmarshal(resp, &response)
	if err3 != nil {
		fmt.Println("Error unmarshaling response:", err3)
		return err3
	}

	fmt.Println("Response received for FT Transfer:", response)

	if !response.Status {
		fmt.Println("error in response for FT: %s", response.Message)
		return fmt.Errorf("error in response for FT: %s", response.Message)
	}

	return err3
}

func (h *DoTransferFTApiCall) callback(
	caller *wasmtime.Caller,
	args []wasmtime.Val,
) ([]wasmtime.Val, *wasmtime.Trap) {
	trieServerSocketConn := h.wasmCtx.ExternalSocketConn()

	// Validate the number of arguments
	inputArgs, outputArgs := utils.HostFunctionParamExtraction(args, true, true)

	// Extract input bytes and convert to string
	inputBytes, memory, err := utils.ExtractDataFromWASM(caller, inputArgs)
	if err != nil {
		fmt.Println("Failed to extract data from WASM", err)
		return utils.HandleError(err.Error())
	}
	h.memory = memory // Assign memory to Host struct for future use
	var transferFTData TransferFTData

	//Unmarshaling the data which has been read from the wasm memory
	err3 := json.Unmarshal(inputBytes, &transferFTData)
	if err3 != nil {
		fmt.Println("Error unmarshaling response in callback function:", err3)
		errMsg := "Error unmarshalling response in callback function" + err3.Error()
		return utils.HandleError(errMsg)
	}
	callTransferFTAPIRespErr := callTransferFTAPI(trieServerSocketConn, h.nodeAddress, h.quorumType, transferFTData)

	if callTransferFTAPIRespErr != nil {
		fmt.Println("failed to transfer NFT", callTransferFTAPIRespErr)
		errMsg := "failed to transfer NFT" + callTransferFTAPIRespErr.Error()
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
