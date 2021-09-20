import PackNFT from 0x{{.PackNFT}}

transaction(commitHash: String, issuer: Address) {
    prepare (minterProxy: AuthAccount) {
        
        let m = minterProxy.borrow<&PackNFT.PDSMinterProxy>(from: PackNFT.minterProxyStoragePath)!
        m.mint(commitHash: commitHash, issuer: issuer)
    } 
}
 
