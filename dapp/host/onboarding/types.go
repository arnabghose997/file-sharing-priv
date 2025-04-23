package onboarding

const ONBOARDING_CONTRACT_ADDRESS = "	"
const DID_DIR = "/home/ubuntu/arnabnode/node4/Rubix/TestNetDID"

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