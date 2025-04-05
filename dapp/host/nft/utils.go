package nft

import (
	"fmt"
	"strings"
)

func extractTransactionIDFromMessage(message string) (string, error) {
	messageElems := strings.Split(message, " ")
	if len(messageElems) == 0 {
		return "", fmt.Errorf("the message is likely empty")
	}

	lastElem := messageElems[len(messageElems) - 1]

	if lastElem == "" {
		return "", fmt.Errorf("transaction ID is empty")
	}

	return lastElem, nil
}