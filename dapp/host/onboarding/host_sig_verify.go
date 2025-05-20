package onboarding

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"

	//"path/filepath"

	"github.com/bytecodealliance/wasmtime-go"

	_ "github.com/joho/godotenv/autoload"
	"github.com/rubixchain/rubix-wasm/go-wasm-bridge/context"
	"github.com/rubixchain/rubix-wasm/go-wasm-bridge/host"
	"github.com/rubixchain/rubix-wasm/go-wasm-bridge/utils"
	rubixCrypto "github.com/rubixchain/rubixgoplatform/crypto"

	"dapp/host/onboarding/store"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

type VerifyAction struct {
	allocFunc   *wasmtime.Func
	memory      *wasmtime.Memory
	nodeAddress string
}

func NewVerifyAction() *VerifyAction {
	return &VerifyAction{}
}

func (h *VerifyAction) Name() string {
	return "do_verify_action"
}

func (h *VerifyAction) FuncType() *wasmtime.FuncType {
	return wasmtime.NewFuncType(
		[]*wasmtime.ValType{
			wasmtime.NewValType(wasmtime.KindI32),
			wasmtime.NewValType(wasmtime.KindI32),
		},
		[]*wasmtime.ValType{wasmtime.NewValType(wasmtime.KindI32)}, // return i32
	)
}

func (h *VerifyAction) Initialize(allocFunc, deallocFunc *wasmtime.Func, memory *wasmtime.Memory, nodeAddress string, quorumType int, wasmCtx *context.WasmContext) {
	h.allocFunc = allocFunc
	h.memory = memory
	h.nodeAddress = nodeAddress
}

func (h *VerifyAction) Callback() host.HostFunctionCallBack {
	return h.callback
}

func (h *VerifyAction) callback(
	caller *wasmtime.Caller,
	args []wasmtime.Val,
) ([]wasmtime.Val, *wasmtime.Trap) {
	_, outputArgs := utils.HostFunctionParamExtraction(args, false, true)

	smartContractInfo, err := getSmartContractInfo(h.nodeAddress, ONBOARDING_CONTRACT_ADDRESS)
	if err != nil {
		errMsg := fmt.Sprintf("failed to get smart contract info, err: %v", err)
		fmt.Println(errMsg)
		return utils.HandleError(errMsg)
	}

	executorDID, err := getSmartContractExecutorDID(smartContractInfo)
	if err != nil {
		fmt.Println("unable to extract executorDID")
		return utils.HandleError(err.Error())
	}

	executorSignature, err := getSmartContractInitiatorSignature(smartContractInfo)
	if err != nil {
		fmt.Println("unable to extract executorSignature")
		return utils.HandleError(err.Error())
	}

	completeDidPath := path.Join(DID_DIR, executorDID, "pubKey.pem")

	executorPubKey, err := getPubKeyFromFile(completeDidPath, executorDID)
	if err != nil {
		errMsg := fmt.Sprintf("failed to load pub key, err: %v", err)
		fmt.Println(errMsg)
		return utils.HandleError(errMsg)
	}

	executorMsg, err := getSmartContractInitiatorSignData(smartContractInfo)
	if err != nil {
		errMsg := fmt.Sprintf("failed to get smart contract block hash, err: %v", err)
		fmt.Println(errMsg)
		return utils.HandleError(errMsg)
	}

	smartContractMsg, err := getSmartContractData(smartContractInfo)
	if err != nil {
		errMsg := fmt.Sprintf("failed to get smart contract msg, err: %v", err)
		fmt.Println(errMsg)
		return utils.HandleError(errMsg)
	}

	providerInfoObj, err := store.UnmarshalSmartContractData(smartContractMsg)
	if err != nil {
		errMsg := fmt.Sprintf("failed to unmarshal smart contract data, err: %v", err)
		fmt.Println(errMsg)
		return utils.HandleError(errMsg)
	}

	if executorDID == providerInfoObj.ProviderDid {
		errMsg := fmt.Sprintf("the executor DID %v is found to be entering their self details, which is not allowed", executorDID)
		fmt.Println(errMsg)
		return utils.HandleError(errMsg)
	}

	isSignatureValid, err := verifyPlatformSignature(executorMsg, executorPubKey, executorSignature)
	if err != nil {
		errMsg := fmt.Sprintf("failed to verify signature, err : %v", err)
		fmt.Println(errMsg)
		return utils.HandleError(errMsg)
	}

	if isSignatureValid {
		err := store.StoreDepinProviderInfo(providerInfoObj)
		if err != nil {
			errMsg := fmt.Sprintf("unable to store Provider Info, err: %v", err)
			fmt.Println(errMsg)
			return utils.HandleError(errMsg)
		}

		responseStr := "Success"
		err = utils.UpdateDataToWASM(caller, h.allocFunc, responseStr, outputArgs)
		if err != nil {
			fmt.Println("Failed to update data to WASM", err)
			return utils.HandleError(err.Error())
		}

		return utils.HandleOk()
	} else {
		responseStr := "Fail"
		err = utils.UpdateDataToWASM(caller, h.allocFunc, responseStr, outputArgs)
		if err != nil {
			fmt.Println("Failed to update data to WASM", err)
			return utils.HandleError(err.Error())
		}

		return utils.HandleOk()
	}
}

func getPubKeyFromFile(path string, did string) (*ecdsa.PublicKey, error) {
	fileObj, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	_, pubKeyByte, err := rubixCrypto.DecodeBIPKeyPair("", nil, fileObj)
	if err != nil {
		return nil, err
	}

	parsedPubKey, err := secp256k1.ParsePubKey(pubKeyByte)
	if err != nil {
		return nil, fmt.Errorf("unable to parse public key.......")
	}

	pubKeySer := parsedPubKey.ToECDSA()

	return pubKeySer, nil
}

func verifyPlatformSignature(message string, pubKey *ecdsa.PublicKey, signature string) (bool, error) {
	messageBytes := []byte(message)

	signatureBytes, err := hex.DecodeString(signature)
	if err != nil {
		return false, err
	}

	return ecdsa.VerifyASN1(pubKey, messageBytes, signatureBytes), nil
}

func getSmartContractInfo(addr string, smartContractHash string) ([]SCTDataReply, error) {
	reqData := map[string]interface{}{
		"token":  smartContractHash,
		"latest": true,
	}
	fmt.Println("Get the contract hash: ", smartContractHash)
	bodyJSON, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	urlFetch, err := url.JoinPath(addr, "/api/get-smart-contract-token-chain-data")
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", urlFetch, bytes.NewBuffer(bodyJSON))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	smartContractResponseStr, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var smartContractResponse SmartContractResponse
	if err := json.Unmarshal(smartContractResponseStr, &smartContractResponse); err != nil {
		return nil, err
	}

	if len(smartContractResponse.SCTDataReply) == 0 {
		return nil, fmt.Errorf("no contract data present")
	} else {
		return smartContractResponse.SCTDataReply, nil
	}
}

func getSmartContractData(smartContractInfo []SCTDataReply) (string, error) {
	latestContractState := smartContractInfo[0]
	return latestContractState.SmartContractData, nil
}

func getSmartContractExecutorDID(smartContractInfo []SCTDataReply) (string, error) {
	latestContractState := smartContractInfo[0]
	return latestContractState.ExecutorDID, nil
}

func getSmartContractInitiatorSignature(smartContractInfo []SCTDataReply) (string, error) {
	latestContractState := smartContractInfo[0]
	return latestContractState.InitiatorSignature, nil
}

func getSmartContractInitiatorSignData(smartContractInfo []SCTDataReply) (string, error) {
	latestContractState := smartContractInfo[0]
	return latestContractState.InitiatorSignData, nil
}
