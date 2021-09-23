import PDS from 0x{{.PDS}}
import ExampleNFT from 0x{{.ExampleNFT}}

transaction (distId: UInt64, nftIDs: [UInt64]) {
    prepare(pds: AuthAccount) {
        let cap = pds.borrow<&PDS.DistributionManager>(from: PDS.distManagerStoragePath) ?? panic("pds does not have Dist manager")
        cap.withdraw(distId: distId, nftIDs: nftIDs, escrowCollectionPublic: ExampleNFT.CollectionPublicPath)
    }
}

