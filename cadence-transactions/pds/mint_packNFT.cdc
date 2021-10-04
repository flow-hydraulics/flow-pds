import PDS from 0x{{.PDS}}
import PackNFT from 0x{{.PackNFT}}
import NonFungibleToken from 0x{{.NonFungibleToken}}

transaction (distId: UInt64, commitHashes: [String], issuer: Address ) {
    prepare(pds: AuthAccount) {
        let recvAcct = getAccount(issuer)
        let recv = recvAcct.getCapability(PackNFT.collectionPublicPath).borrow<&{NonFungibleToken.CollectionPublic}>()
            ?? panic("Unable to borrow Collection Public reference for recipient")
        let cap = pds.borrow<&PDS.DistributionManager>(from: PDS.distManagerStoragePath) ?? panic("pds does not have Dist manager")
        cap.mintPackNFT(distId: distId, commitHashes: commitHashes, issuer: issuer, recvCap: recv)
    }
}

