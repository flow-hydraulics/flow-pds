import PDS from 0x{{.PDS}}
import PackNFT from 0x{{.PackNFT}}
import NonFungibleToken from 0x{{.NonFungibleToken}}

transaction() {
    prepare (issuer: AuthAccount, NFTProviderPath: PrivatePath) {
        
        let i = issuer.borrow<&PDS.PackIssuer>(from: PDS.packIssuerStoragePath) ?? panic ("issuer does not have PackIssuer resource")
        
        // issuer must have a PackNFT collection
        let withdrawCap = issuer.getCapability<&NonFungibleToken.Collection{NonFungibleToken.Provider}>(NFTProviderPath);
        let mintCap = issuer.getCapability<&PackNFT.PackNFTMinter{IPackNFT.IMinter}>(PackNFT.minterPrivPath);
        assert(withdrawCap.check(), message:  "cannot borrow withdraw capability") 
        assert(mintCap.check(), message:  "cannot borrow mint capability") 

        let sc <- createSharedCapabilities (
            withdrawCap: withdrawCap, 
            mintCap: mintCap 
        i.create(sharedCap: <-sc)
    } 
}
 
