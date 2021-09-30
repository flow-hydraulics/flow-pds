import PDS from 0x{{.PDS}}
import ExampleNFT from 0x{{.ExampleNFT}}
import NonFungibleToken from 0x{{.NonFungibleToken}}

transaction (distId: UInt64, packId: UInt64, nftIds: [UInt64], owner: Address) {
    prepare(pds: AuthAccount) {
        let cap = pds.borrow<&PDS.DistributionManager>(from: PDS.distManagerStoragePath) ?? panic("pds does not have Dist manager")
        let recvAcct = getAccount(owner)
        let recv = recvAcct.getCapability(ExampleNFT.CollectionPublicPath).borrow<&{NonFungibleToken.CollectionPublic}>()
            ?? panic("Unable to borrow Collection Public reference for recipient")
        cap.openPackNFT(
            distId: distId,
            packId: packId, 
            nftIds: nftIds, 
            recvCap: recv, 
            collectionProviderPath: ExampleNFT.CollectionProviderPrivPath, 
        )
    }
}

