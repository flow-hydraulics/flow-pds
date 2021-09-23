import PDS from 0x{{.PDS}}
import ExampleNFT from 0x{{.ExampleNFT}}

transaction (distId: UInt64, packId: UInt64, nftIds: [UInt64]) {
    prepare(pds: AuthAccount) {
        let cap = pds.borrow<&PDS.DistributionManager>(from: PDS.distManagerStoragePath) ?? panic("pds does not have Dist manager")
        cap.reveal(distId: distId, packId: packId, nftIDs: nftIDs)
    }
}

