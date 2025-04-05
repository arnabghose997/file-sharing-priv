package nft

type ExtensionCommand struct {
	Action  string                 `json:"action"`  // Specific action to perform (e.g., "sign", "connect", "getAccounts")
	Payload map[string]interface{} `json:"payload"` // Data needed by the extension to execute the command
}
