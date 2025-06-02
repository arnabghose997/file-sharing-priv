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

type CreateFTData struct {
	DID        string `json:"did"`
	FTCount    int32  `json:"ft_count"`
	FTName     string `json:"ft_name"`
	TokenCount int32  `json:"token_count"`
	QuorumType int32  `json:"quorum_type"`
}

type DoCreateFTApiCall struct {
	allocFunc   *wasmtime.Func
	memory      *wasmtime.Memory
	nodeAddress string
	quorumType  int
	wasmCtx     *context.WasmContext
}

func NewDoCreateFTApiCall() *DoCreateFTApiCall {
	return &DoCreateFTApiCall{}
}

func (h *DoCreateFTApiCall) Name() string {
	return "do_create_ft"
}

func (h *DoCreateFTApiCall) FuncType() *wasmtime.FuncType {
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

func (h *DoCreateFTApiCall) Initialize(allocFunc, deallocFunc *wasmtime.Func, memory *wasmtime.Memory, nodeAddress string, quorumType int, wasmContext *context.WasmContext) {
	h.allocFunc = allocFunc
	h.memory = memory
	h.nodeAddress = nodeAddress
	h.quorumType = quorumType
	h.wasmCtx = wasmContext
}

func (h *DoCreateFTApiCall) Callback() host.HostFunctionCallBack {
	return h.callback
}

func callCreateFTAPI(webSocketConn *websocket.Conn, nodeAddress string, quorumType int, createFTdata CreateFTData) error {
	fmt.Println("LOG: call from contract to do Create FT")
	createFTdata.QuorumType = int32(quorumType)

	createFTdataBytes, _ := json.Marshal(createFTdata)
	var createFTDataMap map[string]interface{} = make(map[string]interface{})

	if err := json.Unmarshal(createFTdataBytes, &createFTDataMap); err != nil {
		return fmt.Errorf("error unmarshalling createFTdataBytes: %v", err)
	}

	msgPayload := map[string]interface{}{
		"type": "OPEN_EXTENSION",
		"data": &ExtensionCommand{
			Action:  "CREATE_FT",
			Payload: createFTDataMap,
		},
	}

	msgPayloadBytes, _ := json.Marshal(msgPayload)

	err := webSocketConn.WriteMessage(websocket.TextMessage, msgPayloadBytes)
	if err != nil {
		return fmt.Errorf("error occured while invoking FT create, err: %v", err)
	}

	_, resp, err := webSocketConn.ReadMessage()
	if err != nil {
		return fmt.Errorf("unable to read response from web socket connection for FT Create, err: %v", err)
	}

	var response *BasicResponse
	err3 := json.Unmarshal(resp, &response)
	if err3 != nil {
		fmt.Println("Error unmarshaling response:", err3)
		return err3
	}

	if !response.Status {
		fmt.Printf("error in response for FT: %s\n", response.Message)
		return fmt.Errorf("error in response for FT: %s", response.Message)
	}

	return nil
}

func (h *DoCreateFTApiCall) callback(
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
	var createFTData CreateFTData

	//Unmarshaling the data which has been read from the wasm memory
	err3 := json.Unmarshal(inputBytes, &createFTData)
	if err3 != nil {
		fmt.Println("Error unmarshaling response in callback function:", err3)
		errMsg := "Error unmarshalling response in callback function" + err3.Error()
		return utils.HandleError(errMsg)
	}
	callCreateFTAPIRespErr := callCreateFTAPI(trieServerSocketConn, h.nodeAddress, h.quorumType, createFTData)

	if callCreateFTAPIRespErr != nil {
		fmt.Println("failed to create FT", callCreateFTAPIRespErr)
		errMsg := "failed to create FT" + callCreateFTAPIRespErr.Error()
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
