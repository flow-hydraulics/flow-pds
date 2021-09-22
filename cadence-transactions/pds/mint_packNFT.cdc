import PDS from 0x{{.PDS}}

transaction (distId: UInt64, commitHashes: [String], issuer: Address ) {
    prepare(pds: AuthAccount) {
        let cap = pds.borrow<&PDS.DistributionManager>(from: PDS.distManagerStoragePath) ?? panic("pds does not have Dist manager")
        cap.mintPackNFT(distId: distId, commitHashes: commitHashes, issuer: issuer)
    }
}

