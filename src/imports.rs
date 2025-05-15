extern "C" {
    pub fn do_mint_nft_trie(
        inputdata_ptr: *const u8,
        inputdata_len: usize,
        resp_ptr_ptr: *mut *const u8,
        resp_len_ptr: *mut usize,
    ) -> i32;

    pub fn do_transfer_ft_trie(
        inputdata_ptr: *const u8,
        inputdata_len: usize,
        resp_ptr_ptr: *mut *const u8,
        resp_len_ptr: *mut usize,
    ) -> i32;

    pub fn do_create_ft(
        inputdata_ptr: *const u8,
        inputdata_len: usize,
        resp_ptr_ptr: *mut *const u8,
        resp_len_ptr: *mut usize,
    ) -> i32;
}