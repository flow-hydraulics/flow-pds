import PDS from 0x{{.PDS}}
import ExampleNFT from 0x{{.ExampleNFT}}

transaction (distId: UInt64, packId: UInt64, nftContractAddrs: [Address], nftContractName: [String], nftIds: [UInt64], salt: String) {
    prepare(pds: AuthAccount) {
        let cap = pds.borrow<&PDS.DistributionManager>(from: PDS.distManagerStoragePath) ?? panic("pds does not have Dist manager")
        //  revealPackNFT(packId: UInt64, nftContractAddrs: [Address], nftContractName: [String], nftIds: [UInt64], salt: String) {
        cap.revealPackNFT(
            distId: distId, 
            packId: packId, 
            nftContractAddrs: nftContractAddrs, 
            nftContractName: nftContractName, 
            nftIds: nftIds, 
            salt: salt)
    }
}

