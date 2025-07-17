package credits

import (
	"encoding/json"
	"fmt"

	"github.com/bytecodealliance/wasmtime-go"
	"github.com/rubixchain/rubix-wasm/go-wasm-bridge/context"
	"github.com/rubixchain/rubix-wasm/go-wasm-bridge/host"
	"github.com/rubixchain/rubix-wasm/go-wasm-bridge/utils"
)

type DoAddCredit struct {
	allocFunc *wasmtime.Func
	memory    *wasmtime.Memory
}

func NewDoAddCredit() *DoAddCredit {
	return &DoAddCredit{}
}

func (h *DoAddCredit) Name() string {
	return "do_add_credit"
}

func (h *DoAddCredit) FuncType() *wasmtime.FuncType {
	return wasmtime.NewFuncType(
		[]*wasmtime.ValType{
			wasmtime.NewValType(wasmtime.KindI32), // input_ptr
			wasmtime.NewValType(wasmtime.KindI32), // input_len
		},
		[]*wasmtime.ValType{wasmtime.NewValType(wasmtime.KindI32)}, // return i32
	)
}

func (h *DoAddCredit) Initialize(allocFunc, deallocFunc *wasmtime.Func, memory *wasmtime.Memory, nodeAddress string, quorumType int, wasmContext *context.WasmContext) {
	h.allocFunc = allocFunc
	h.memory = memory
}

func (h *DoAddCredit) Callback() host.HostFunctionCallBack {
	return h.callback
}

type AddCreditData struct {
	UserDid string  `json:"user_did"`
	Credit  float64 `json:"credit"`
}

func (h *DoAddCredit) callback(caller *wasmtime.Caller, args []wasmtime.Val) ([]wasmtime.Val, *wasmtime.Trap) {
	inputArgs, _ := utils.HostFunctionParamExtraction(args, true, false)

	// Extract input bytes
	inputBytes, memory, err := utils.ExtractDataFromWASM(caller, inputArgs)
	if err != nil {
		fmt.Println("Failed to extract data from WASM", err)
		return utils.HandleError(err.Error())
	}
	h.memory = memory // Assign memory to Host struct for future use

	var addCreditData AddCreditData
	//Unmarshaling the data which has been read from the wasm memory
	err = json.Unmarshal(inputBytes, &addCreditData)
	if err != nil {
		fmt.Println("Error unmarshaling response in callback function:", err)
		errMsg := "Error unmarshaling response in callback function:" + err.Error()
		return utils.HandleError(errMsg)
	}

	return utils.HandleOk()
}
