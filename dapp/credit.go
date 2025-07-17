package main

import (
	"dapp/host/ft"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	wasmbridge "github.com/rubixchain/rubix-wasm/go-wasm-bridge"
	wasmContext "github.com/rubixchain/rubix-wasm/go-wasm-bridge/context"
	"github.com/syndtr/goleveldb/leveldb"

	"dapp/host/credits"
)

type CreditInfo struct {
	Credit    uint   `json:"credit"`
	Timestamp string `json:"timestamp"`
}

func (s *Server) handleGetCreditBalance(c *gin.Context) {
	w := http.ResponseWriter(c.Writer)
	enableCors(&w)

	did := c.Param("did")
	if did == "" {
		getClientError(c, "DID is required")
		return
	}

	if s.DB == nil {
		getInternalError(c, "Database is not initialized")
		return
	}

	// Assuming GetCreditBalance is a function that retrieves the credit balance for the given DID
	balance, err := getCreditBalance(s.DB, did)
	if err != nil {
		getInternalError(c, "Failed to retrieve credit balance: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"did": did, "credit ": balance})
}

func getCreditBalance(db *leveldb.DB, did string) (*CreditInfo, error) {
	creditInfoBytes, err := db.Get([]byte(did), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return &CreditInfo{Credit: 0, Timestamp: ""}, nil
		}
		return nil, fmt.Errorf("failed to get credit balance for DID %s: %v", did, err)
	}

	var creditInfo *CreditInfo
	err = json.Unmarshal(creditInfoBytes, &creditInfo)
	if err != nil {
		return nil, err // Error unmarshaling the balance
	}

	return creditInfo, nil
}

type AddCredit struct {
	UserDid string  `json:"user_did"`
	Credit  float64 `json:"credit"`
}

func (s *Server) handleAddCredits(c *gin.Context) {
	nodeAddress := "http://localhost:20007"
	quorumType := 2

	selfContractHashPath := path.Join("../artifacts/inference_credit_purchase_contract.wasm")

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
	hostFnRegistry.Register(credits.NewDoAddCredit())

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

	creditInfoStr, err := wasmModule.CallFunction(contractInputRequest.SmartContractData)
	if err != nil {
		wrapError(c.JSON, fmt.Sprintf("unable to execute function, err: %v", err))
		return
	}

	var addCredit AddCredit
	err = json.Unmarshal([]byte(creditInfoStr), &addCredit)
	if err != nil {
		wrapError(c.JSON, fmt.Sprintf("unable to unmarshal credit info, err: %v", err))
		return
	}

	currTimestamp := strconv.FormatInt(time.Now().Unix(), 10)

	// Assuming AddCredits is a function that adds credits to the given DID
	err = addCreditsToDB(s.DB, addCredit.UserDid, uint(addCredit.Credit), currTimestamp)
	if err != nil {
		getInternalError(c, "Failed to add credits: "+err.Error())
		return
	}

	wrapSuccess(c.JSON, fmt.Sprintf("Successfully added %.2f credits to DID %s", addCredit.Credit, addCredit.UserDid))
	return
}

type DeductCreditsReq struct {
	DID string `json:"did"`
}

func (s *Server) handleDeductCredits(c *gin.Context) {
	w := http.ResponseWriter(c.Writer)
	enableCors(&w)

	var deductCreditsReq DeductCreditsReq
	err := json.NewDecoder(c.Request.Body).Decode(&deductCreditsReq)
	if err != nil {
		wrapError(c.JSON, "err: Invalid request body")
		return
	}
	if deductCreditsReq.DID == "" {
		getClientError(c, "DID is required")
		return
	}

	err = deductCreditsFromDB(s.DB, deductCreditsReq.DID)
	if err != nil {
		getInternalError(c, "Failed to deduct credits: "+err.Error())
		return
	}

	wrapSuccess(c.JSON, fmt.Sprintf("Successfully deducted credits from DID %s", deductCreditsReq.DID))
}

func deductCreditsFromDB(db *leveldb.DB, did string) error {
	getCreditBalance, err := getCreditBalance(db, did)
	if err != nil {
		return fmt.Errorf("failed to get existing credit balance for DID %s: %v", did, err)
	}

	getCreditBalance.Credit -= 1
	getCreditBalance.Timestamp = strconv.FormatInt(time.Now().Unix(), 10)

	creditInfoBytes, err := json.Marshal(getCreditBalance)
	if err != nil {
		return fmt.Errorf("failed to marshal credit info: %v", err)
	}

	err = db.Put([]byte(did), creditInfoBytes, nil)
	if err != nil {
		return fmt.Errorf("failed to deduct credits for DID %s: %v", did, err)
	}

	return nil
}

func addCreditsToDB(db *leveldb.DB, did string, creditCount uint, currTimestamp string) error {
	getCreditBalance, err := getCreditBalance(db, did)
	if err != nil {
		return fmt.Errorf("failed to get existing credit balance for DID %s: %v", did, err)
	}

	if getCreditBalance.Credit == 0 {
		getCreditBalance.Credit = creditCount
		getCreditBalance.Timestamp = currTimestamp
	} else {
		getCreditBalance.Credit += creditCount
		getCreditBalance.Timestamp = currTimestamp
	}

	// For example, you might want to update the existing credit balance or create a new entry.
	creditInfoBytes, err := json.Marshal(getCreditBalance)
	if err != nil {
		return fmt.Errorf("failed to marshal credit info: %v", err)
	}

	err = db.Put([]byte(did), creditInfoBytes, nil)
	if err != nil {
		return fmt.Errorf("failed to add credits for DID %s: %v", did, err)
	}

	return nil
}
