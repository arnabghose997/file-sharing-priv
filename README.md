# Environment variable setup

Refer `dapp/.env.sample` and create another `.env` file under `dapp` directory. The `RUBIX_NFT_DIR` mentions the complete path `NFT` directory present under your Rubix node directory. For instance, if your node folder is `node5`, the `NFT` directory is present under `node5/NFT`.

# Artifact Upload and Fetch Server

Following endpoints are added to facilitate the upload of NFT Artifact and Metadata, as well as fetching of NFT Artifact

1. POST: `/api/upload_asset/upload_artifacts` (Uploads both NFT Artifact and NFT Metadata, and stores them in the `./dapp/uploads` dir)

    - Request Type: `form-data`
    - Params:
        - `asset (File)`: Pass NFT Artifact file here
        - `metadata (File)`: Pass NFT Metadata here
    - Example (Request):
        ```bash
        curl --location --request POST 'http://localhost:8082/api/upload_asset/upload_artifacts' \
        --form 'asset=@"<location of asset file>"' \
        --form 'metadata=@"<location of metadata.json file>"'
        ```
    - Example (Response):
        - Success:
        ```json
        {
            "artifactPath": "uploads/1744161744/asset.txt",
            "metadataPath": "uploads/1744161744/metadata.json",
            "status": true
        }
        ```
        - Fail (skipped adding either of the two files):
        ```
        {
            "error": "Failed to get metadata file, metadata file is required",
            "status": false
        }
        ```

2. GET: `/api/upload_asset/get_artifact_info_by_cid/:<nftId>` (Retrieves the `metadata.json` content for a particular NFT ID in base64 encoding. This is essential for displaying the assets (AI model or Dataset) owned by a DID)

    - Params:
        - `nftId`: Pass the NFT ID here
         
    - Example (Request):
        ```bash
        curl --location --request GET 'http://localhost:8082/api/upload_asset/get_artifact_info_by_cid/QmAb123'
        ```
    - Example (Response):
        - Success:
        ```json
        {
            "artifactMetadata": "eyJkZXNjcmlwdGlvbiI6ImRlc2MiLCJuYW1lIjoiRGF0YXNldCJ9",
            "status": true
        }
        ```
        - Fail (invalid NFT ID):
        ```json
        {
            "error": "failed to read asset metadata file: open \\windows\\node9\\NFT/QmAb1234/metadata.json: The system cannot find the path specified.",
            "status": false
        }
        ```

3. GET: `/api/upload_asset/get_artifact_file_name/:<nftID>` - Gets the name of artifact file for an NFT

    - Params:
        - `nftId`: Pass the NFT ID here
         
    - Example (Request):
        ```bash        
        curl --location --request GET 'http://localhost:8082/api/upload_asset/get_artifact_file_name/QmAb123' --header 'Content-Type: application/json'
        ```
    - Example (Response):
        - Success:
        ```json
        {
            "artifactFileName": "metadata.exe",
            "status": true
        }
        ```
        - Fail (invalid NFT ID):
        ```json
        {
            "error": "no artifact file found for NFT ID QmAb123",
            "status": false
        }
        ```

# Asset Publish Contract

## Setup

1. /api/register-callback-url

The url provided here would: `http://localhost:<port of dapp server>/api/upload_asset`

## Smart Contract Input

Following is the format for Smart Contract input:

```json
"publish_asset": {
    "asset_artifact": <path to AI model or Dataset file>,
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

# Asset Usage Contract

## Setup

1. /api/register-callback-url

The url provided here would: `http://localhost:<port of dapp server>/api/use_asset`

## Smart Contract Input

Following is the format for Smart Contract input:

```json
"use_asset": {
    "asset_usage_price": "(Whole Value, int) orignal value of NFT in TRIE",
    "asset_user_did": "Xell connected DID",
    "asset_usage_purpose": "Description string mentioning the intent of the action. In this case, we can write `AI Model/Dataset bought by <asset_user_did>",
    "asset_denom": "TRIE",
    "asset_owner_did": "The original owner of the NFT",
    "asset_id": "NFT ID",
    "asset_value": "(float) orignal value of NFT in RBT",

    "ft_denom_creator": "DID of the creator of TRIE token",
}
```

NOTE: It should stringified before passed in the `/api/execute-smart-contract`

