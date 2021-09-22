import PDS from 0x{{.PDS}}
import PackNFT from 0x{{.PackNFT}}
import IPackNFT from 0x{{.IPackNFT}}
import NonFungibleToken from 0x{{.NonFungibleToken}}

transaction(NFTProviderPath: PrivatePath) {
    prepare (issuer: AuthAccount) {
        
        let i = issuer.borrow<&PDS.PackIssuer>(from: PDS.packIssuerStoragePath) ?? panic ("issuer does not have PackIssuer resource")
        
        // issuer must have a PackNFT collection
        log(NFTProviderPath)
        let withdrawCap = issuer.getCapability<&{NonFungibleToken.Provider}>(NFTProviderPath);
        let mintCap = issuer.getCapability<&{IPackNFT.IMinter}>(PackNFT.minterPrivPath);
        assert(withdrawCap.check(), message:  "cannot borrow withdraw capability") 
        assert(mintCap.check(), message:  "cannot borrow mint capability") 

        let sc <- PDS.createSharedCapabilities ( withdrawCap: withdrawCap, mintCap: mintCap )
        i.create(sharedCap: <-sc)
    } 
}
 
