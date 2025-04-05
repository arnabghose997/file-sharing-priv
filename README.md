# Asset Publish Contract

## Setup

1. /api/register-callback-url

The url provided here would: `http://localhost:<port of dapp server>/api/upload_asset`

## Smart Contract Input

Following is the format for Smart Contract input:

```json
"publish_asset": {
    "asset_artifiact": <path to AI model or Dataset file>,
    "asset_metadata": <JSON file containing metadata information about AI model or Dataset>,
    "asset_owner_did": DID from Connected Xell Wallet,
    "asset_publish_description": Description string mentioning the intent of the action. In this case, we can write `AI Model/Dataset published and owned by <owner_did>`,
    "asset_value": Value of the Asset. Here, the value will be in RBT. From the frontend, the value expected from User will be in TRIE, which is then supposed to converted to equivalent value in RBT,
    "depin_provider_did": DID of the DePIN provider,
    "depin_hosting_cost": Value of the Asset. This would in TRIE only, // Hosting fees 

    "tx_denom": The value should be `TRIE`
}
```

NOTE: It should stringified before passed in the `/api/execute-smart-contract`