import PackNFT from 0x{{.PackNFT}}

transaction() {
    prepare (pds: AuthAccount) {
        
        // Check if account already have a minterproxy resource, if so destroy it
        if pds.borrow<&PackNFT.PDSMinterProxy>(from: PackNFT.minterProxyStoragePath) != nil {
            pds.unlink(PackNFT.minterProxyMintCapRecv)
            let p <- pds.load<@PackNFT.PDSMinterProxy>(from: PackNFT.minterProxyStoragePath) 
            destroy p
        }
        
        pds.save(<- PackNFT.createNewMinterProxy(), to: PackNFT.minterProxyStoragePath);
        
        pds.link<&PackNFT.PDSMinterProxy{PackNFT.MintCapReceiver}>(PackNFT.minterProxyMintCapRecv, target: PackNFT.minterProxyStoragePath)
        ??  panic("Could not link MintCapReceiver");
    } 
}
 
