import PDS from 0x{{.PDS}}
import {{.PackNFTName}} from 0x{{.PackNFTAddress}}
import IPackNFT from 0x{{.IPackNFT}}
import NonFungibleToken from 0x{{.NonFungibleToken}}

transaction(NFTProviderPath: PrivatePath, title: String, metadata: {String: String}) {
    prepare (issuer: AuthAccount) {

        let i = issuer.borrow<&PDS.PackIssuer>(from: PDS.PackIssuerStoragePath) ?? panic ("issuer does not have PackIssuer resource")

        // issuer must have a PackNFT collection
        log(NFTProviderPath)
        let withdrawCap = issuer.getCapability<&{NonFungibleToken.Provider}>(NFTProviderPath);
        let operatorCap = issuer.getCapability<&{IPackNFT.IOperator}>({{.PackNFTName}}.OperatorPrivPath);
        assert(withdrawCap.check(), message:  "cannot borrow withdraw capability")
        assert(operatorCap.check(), message:  "cannot borrow operator capability")

        let sc <- PDS.createSharedCapabilities ( withdrawCap: withdrawCap, operatorCap: operatorCap )
        i.create(sharedCap: <-sc, title: title, metadata: metadata)
    }
}
