import PDS from 0x{{.PDS}}
import PackNFT from 0x{{.PackNFT}}
import ExampleNFT from 0x{{.ExampleNFT}}
import NonFungibleToken from 0x{{.NonFungibleToken}}

transaction (distId: UInt64, packId: UInt64, nftContractAddrs: [Address], nftContractName: [String], nftIds: [UInt64], salt: String, owner: Address, openRequest: Bool) {
    prepare(pds: AuthAccount) {
        let cap = pds.borrow<&PDS.DistributionManager>(from: PDS.distManagerStoragePath) ?? panic("pds does not have Dist manager")
        let p = PackNFT.borrowPackRepresentation(id: packId) ?? panic ("No such pack")
        if openRequest && p.status == PackNFT.Status.Revealed {
            let recvAcct = getAccount(owner)
            let recv = recvAcct.getCapability(ExampleNFT.CollectionPublicPath).borrow<&{NonFungibleToken.CollectionPublic}>()
                ?? panic("Unable to borrow Collection Public reference for recipient")
            cap.openPackNFT(
                distId: distId,
                packId: packId, 
                nftContractAddrs: nftContractAddrs, 
                nftContractName: nftContractName, 
                nftIds: nftIds, 
                recvCap: recv, 
                collectionProviderPath: ExampleNFT.CollectionProviderPrivPath, 
            )
        } else {
            cap.revealPackNFT(
                    distId: distId, 
                    packId: packId, 
                    nftContractAddrs: nftContractAddrs, 
                    nftContractName: nftContractName, 
                    nftIds: nftIds, 
                    salt: salt)
        }
    }
}

