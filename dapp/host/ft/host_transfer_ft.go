package ft

import (
	"encoding/json"
	"fmt"
	"time"

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
	transferFTdata.QuorumType = int32(quorumType)

	transferFTdataBytes, _ := json.Marshal(transferFTdata)
	var transferFTDataMap map[string]interface{} = make(map[string]interface{})

	if err := json.Unmarshal(transferFTdataBytes, &transferFTDataMap); err != nil {
		return fmt.Errorf("error unmarshalling transferFTdataBytes: %v", err)
	}

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
		return fmt.Errorf("error occured while invoking FT Transfer, err: %v", err)
	}
	// bodyJSON, err := json.Marshal(transferFTdata)
	// if err != nil {
	// 	fmt.Println("Error marshaling JSON:", err)
	// 	return err
	// }

	// transferFTUrl, err := url.JoinPath(nodeAddress, "/api/initiate-ft-transfer")
	// if err != nil {
	// 	return err
	// }

	// req, err := http.NewRequest("POST", transferFTUrl, bytes.NewBuffer(bodyJSON))
	// if err != nil {
	// 	fmt.Println("Error creating HTTP request:", err)
	// 	return err
	// }

	// req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	// client := &http.Client{}
	// resp, err := client.Do(req)
	// if err != nil {
	// 	fmt.Println("Error sending HTTP request:", err)
	// 	return err
	// }
	// defer resp.Body.Close()
	errDeadline := webSocketConn.SetReadDeadline(time.Now().Add(5 * time.Minute))
	if errDeadline != nil {
		return fmt.Errorf("error setting read deadline for web socket connection, err: %v", err)
	}

	_, resp, err := webSocketConn.ReadMessage()
	if err != nil {
		return fmt.Errorf("unable to read response from web socket connection for FT Transfer, err: %v", err)
	}

	var response map[string]interface{}
	err3 := json.Unmarshal(resp, &response)
	if err3 != nil {
		fmt.Println("Error unmarshaling response:", err3)
		return err3
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
