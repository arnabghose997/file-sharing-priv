pub mod helpers;
pub mod imports;

use std::fmt::format;

use helpers::{call_mint_nft_api, MintNft, TransferFt, call_transfer_ft_api, CreateFt, call_create_ft_api};

use rubixwasm_std::errors::WasmError;
use serde::{Deserialize, Serialize};
use rubixwasm_std::contract_fn;

#[derive(Serialize, Deserialize)]
pub struct CreateTokenRequest {
    pub creator_did: String,
    pub token_name: String,
    pub total_supply: u32,
    pub rbt_amount: u32,
}

#[derive(Serialize, Deserialize)]
pub struct PublishAssetReq {
    asset_artifact: String,
    asset_metadata: String,
    asset_owner_did: String,
    asset_publish_description: String,
    asset_value: String,

    depin_provider_did: String,
    depin_hosting_cost: u32, // Hosting fees 

    ft_denom: String,
    ft_denom_creator: String
}

#[contract_fn]
pub fn publish_asset(publish_asset_req: PublishAssetReq) -> Result<String, WasmError> {
    // Create NFT for AI Model/Dataset
    
    let asset_value_float: f64 = match publish_asset_req.asset_value.parse::<f64>() {
        Ok(value) => value,
        Err(_) => return Err(WasmError::from(format!("failed to parse asset_value: {}", publish_asset_req.asset_value))),
    };

    let asset_creation_req = MintNft {
        did: publish_asset_req.asset_owner_did.clone(),
        metadata: publish_asset_req.asset_metadata.clone(),
        artifact: publish_asset_req.asset_artifact,
        nftData: publish_asset_req.asset_publish_description,
        nftValue: asset_value_float,
    };

    let mint_nft_response = match call_mint_nft_api(asset_creation_req) {
        Ok(res) => res,
        Err(e) => return Err(WasmError::from(format!("failed while calling call_mint_nft_api, err: {:?}", e))),
    };

    let nft_id: String = mint_nft_response.nftId;

    // Pay Depin Provider in TRIE, and mention the NFT ID in the `comment`
    // for them to fetch NFT`
    let depin_payment_req = TransferFt {
        comment: format!("nft:{}", nft_id),
        ft_count: publish_asset_req.depin_hosting_cost as i32,
        ft_name: publish_asset_req.ft_denom,
        creatorDID: publish_asset_req.ft_denom_creator,
        sender: publish_asset_req.asset_owner_did,
        receiver: publish_asset_req.depin_provider_did.clone()
    };

    match call_transfer_ft_api(depin_payment_req) {
        Ok(_) => return Ok("".to_string()),
        Err(_) => return Err(WasmError { msg: format!("failed to send TRIE to DePin provider {}, please use 'resend_hosting_fees' contract function to retry sending TRIE tokens", publish_asset_req.depin_provider_did) }),
    };
}


#[derive(Serialize, Deserialize)]
pub struct ResendHostingFeesReq {
    depin_hosting_cost: u32,
    depin_provider_did: String,
    asset_owner_did: String,
    asset_id: String, // NFT ID of the Asset

    tx_denom: String
}

// Resend TRIE tokens to provider if the TRIE transaction failed while calling publish_asset  
#[contract_fn]
pub fn resend_hosting_fees(resend_hosting_fees_req: ResendHostingFeesReq) -> Result<String, WasmError> {
    let depin_payment_req = TransferFt {
        comment: format!("nft:{}", resend_hosting_fees_req.asset_id),
        ft_count: resend_hosting_fees_req.depin_hosting_cost as i32,
        ft_name: resend_hosting_fees_req.tx_denom,
        creatorDID: "".to_string(),
        sender: resend_hosting_fees_req.asset_owner_did,
        receiver: resend_hosting_fees_req.depin_provider_did.clone()
    };

    match call_transfer_ft_api(depin_payment_req) {
        Ok(_) => return Ok("".to_string()),
        Err(_) => return Err(WasmError { msg: format!("failed to send TRIE to DePin provider {}, please use 'resend_hosting_fees' contract function to retry sending TRIE tokens", resend_hosting_fees_req.depin_provider_did) }),
    };
}

#[contract_fn]
pub fn create_new_token(request: CreateTokenRequest) -> Result<String, WasmError> {
    // Validate input parameters
    if request.creator_did.is_empty() {
        return Err(WasmError::from("Creator DID cannot be empty"));
    }

    if request.token_name.is_empty() {
        return Err(WasmError::from("Token name cannot be empty"));
    }

    if request.total_supply == 0 {
        return Err(WasmError::from("Total supply must be greater than 0"));
    }

    if request.rbt_amount == 0 {
        return Err(WasmError::from("RBT amount must be greater than 0"));
    }

    // Prepare data for CREATE_FT host function
    let create_ft_data = CreateFt {
        did: request.creator_did,
        ft_count: request.total_supply as i32,
        ft_name: request.token_name,
        token_count: request.rbt_amount as i32,
    };

    // Call the CREATE_FT host function
    match call_create_ft_api(create_ft_data) {
        Ok(response) => {
            Ok(format!("Token created successfully. Transaction ID: {}", response.txId))
        },
        Err(e) => {
            Err(WasmError::from(format!("Failed to create token: {:?}", e)))
        }
    }
}