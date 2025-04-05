use super::imports::{do_mint_nft_trie, do_transfer_ft_trie};
use std::str;
use serde::{Serialize,Deserialize};
use rubixwasm_std::errors::WasmError;
use std::slice;

#[derive(Serialize, Deserialize)]
pub struct MintNft {
    pub did:      String, 
    pub metadata: String,
    pub artifact: String,
    pub nftData:  String,
    pub nftValue: i32,
}

#[derive(Serialize, Deserialize)]
pub struct MintNftResponse {
    pub nftId: String,
    pub txId: String
}

#[derive(Serialize, Deserialize)]
pub struct TransferFt{
    pub comment: String, 
    pub ft_count: i32,
    pub ft_name: String,
    pub creatorDID: String,
    pub sender: String,
    pub receiver: String,
}

pub fn call_mint_nft_api(mint_nft: MintNft) -> Result<MintNftResponse, WasmError> {
    unsafe {
        // Convert the input data to bytes
        let input_bytes = serde_json::to_string(&mint_nft).unwrap().into_bytes();

        // let input_bytes = input_data.as_bytes();
        let input_ptr = input_bytes.as_ptr();
        let input_len = input_bytes.len();

        // Allocate space for the response pointer and length
        let mut resp_ptr: *const u8 = std::ptr::null();
        let mut resp_len: usize = 0;

        // Call the imported host functionrubixwasm_std::
        let result = do_mint_nft_trie(
            input_ptr,
            input_len,
            &mut resp_ptr,
            &mut resp_len,
        );
        
        if result != 0 {
            return Err(WasmError::from(format!("Host function returned error code {}", result)));
        }

        // Ensure the response pointer is not null
        if resp_ptr.is_null() {
            return Err(WasmError::from("Response pointer is null".to_string()));
        }

        // Convert the response back to a Rust String
        let response_slice = slice::from_raw_parts(resp_ptr, resp_len);
        match str::from_utf8(response_slice) {
            Ok(s) => {
                let resp_string = s.to_string();
                let mint_nft_response: MintNftResponse = serde_json::from_str(&resp_string).unwrap();
                Ok(mint_nft_response)
            },
            Err(_) => Err(WasmError::from("Invalid UTF-8 response".to_string())),
        }
    }
}


pub fn call_transfer_ft_api(input_data: TransferFt) -> Result<String, WasmError> {
    unsafe {
        // Convert the input data to bytes
        let input_bytes = serde_json::to_string(&input_data).unwrap().into_bytes();

        // let input_bytes = input_data.as_bytes();
        let input_ptr = input_bytes.as_ptr();
        let input_len = input_bytes.len();

        // Allocate space for the response pointer and length
        let mut resp_ptr: *const u8 = std::ptr::null();
        let mut resp_len: usize = 0;

        // Call the imported host functionrubixwasm_std::
        let result = do_transfer_ft_trie(
            input_ptr,
            input_len,
            &mut resp_ptr,
            &mut resp_len,
        );
        
        if result != 0 {
            return Err(WasmError::from(format!("Host function returned error code {}", result)));
        }

        // Ensure the response pointer is not null
        if resp_ptr.is_null() {
            return Err(WasmError::from("Response pointer is null".to_string()));
        }

        // Convert the response back to a Rust String
        let response_slice = slice::from_raw_parts(resp_ptr, resp_len);
        match str::from_utf8(response_slice) {
            Ok(s) => Ok(s.to_string()),
            Err(_) => Err(WasmError::from("Invalid UTF-8 response".to_string())),
        }
    }

}