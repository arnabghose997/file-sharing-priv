package onboarding

const ONBOARDING_CONTRACT_ADDRESS = "QmWGd62Mt82YwaVmHwLnRcWsVmnruPKPkd42BfuDwopkYt"
const DID_DIR = "/home/ubuntu/arnabnode/node7/Rubix/TestNetDID"

type SmartContractResponse struct {
	BasicResponse
	SCTDataReply []SCTDataReply
}

type BasicResponse struct {
	Status  bool        `json:"status"`
	Message string      `json:"message"`
	Result  interface{} `json:"result"`
}

type SCTDataReply struct {
	BlockNo            uint64
	BlockId            string
	SmartContractData  string
	Epoch              int
	InitiatorSignature string
	ExecutorDID        string
	InitiatorSignData  string
}