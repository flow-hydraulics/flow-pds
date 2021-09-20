import PackNFT from 0x{{.PackNFT}}
import IPackNFT from 0x{{.IPackNFT}}

transaction (pds: Address) {
    prepare(issuer: AuthAccount) {
        log(issuer)
        log(pds)
        let cap = issuer.getCapability<&PackNFT.PackNFTMinter{IPackNFT.IMinter}>(PackNFT.minterPrivPath);
        if !cap.check() {
            panic ("cannot borrow such capability") 
        } else {
            let setCapRef = getAccount(pds).getCapability<&PackNFT.PDSMinterProxy{PackNFT.MintCapReceiver}>(PackNFT.minterProxyMintCapRecv).borrow() ?? panic("Cannot get receiver");
            setCapRef.setMintCap(mintCap: cap);
        }
    }

}

