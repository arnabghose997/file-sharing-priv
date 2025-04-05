package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/joho/godotenv/autoload"
	_ "github.com/mattn/go-sqlite3"
)

type TxInfo struct {
	Comment string
}

func doIPFSGet(hash string) error { return nil }
func doIPFSPin(hash string) error { return nil }

func extractNFTIDFromComment(comment string) (string, error) {
	// Split the comment by ":"
	parts := strings.Split(comment, ":")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid comment format: %s", comment)
	}

	// Return the NFT ID
	return parts[1], nil
}

func FetchAndPinFTs() error {
	txs, err := fetchTxsWithNFTFetchAndPinIntentions()
	if err != nil {
		return err
	}

	for _, tx := range txs {
		nftId, err := extractNFTIDFromComment(tx.Comment)
		if err != nil {
			return fmt.Errorf("failed to extract NFT ID from comment: %w", err)
		}

		errIPFSGet := doIPFSGet(nftId)
		if errIPFSGet != nil {
			return fmt.Errorf("failed to fetch NFT from IPFS: %w", errIPFSGet)
		}

		errIPFSPin := doIPFSPin(nftId)
		if errIPFSPin != nil {
			return fmt.Errorf("failed to pin NFT on IPFS: %w", errIPFSPin)
		}

		AddNFTRecordToHostingListInfo(nftId)
	}
	return nil
}

func fetchTxsWithNFTFetchAndPinIntentions() ([]*TxInfo, error){
	// Open database connection
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		return nil, fmt.Errorf("failed to get DB_PATH")
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Execute query
	ftTransactionsForNFTFetch, err := db.Query("SELECT comment FROM TransactionHistory WHERE comment LIKE 'nft:%'")
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer ftTransactionsForNFTFetch.Close()

	// Process results
	var txs []*TxInfo = make([]*TxInfo, 0)
	for ftTransactionsForNFTFetch.Next() {
		var tx *TxInfo
		if err := ftTransactionsForNFTFetch.Scan(&tx.Comment); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		txs = append(txs, tx)
	}

	// Check for errors after iteration
	if err = ftTransactionsForNFTFetch.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return txs, nil
}