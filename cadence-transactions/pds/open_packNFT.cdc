import PDS from 0x{{.PDS}}
import ExampleNFT from 0x{{.ExampleNFT}}

transaction (distId: UInt64, packId: UInt64, nftIds: [UInt64], owner: Address) {
    prepare(pds: AuthAccount) {
        let cap = pds.borrow<&PDS.DistributionManager>(from: PDS.distManagerStoragePath) ?? panic("pds does not have Dist manager")
        cap.openPackNFT(
            distId: distId,
            packId: packId, 
            nftIds: nftIds, 
            owner: owner, 
            collectionProviderPath: ExampleNFT.CollectionProviderPrivPath, 
            recvCollectionPublicPath: ExampleNFT.CollectionPublicPath
        )
    }
}

