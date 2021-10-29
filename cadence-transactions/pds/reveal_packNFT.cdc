import PDS from 0x{{.PDS}}
import {{.PackNFTName}} from 0x{{.PackNFTAddress}}
import {{.CollectibleNFTName}} from 0x{{.CollectibleNFTAddress}}
import NonFungibleToken from 0x{{.NonFungibleToken}}

transaction (
    distId: UInt64,
    packId: UInt64,
    nftContractAddrs: [Address],
    nftContractName: [String],
    nftIds: [UInt64],
    salt: String,
    owner: Address,
    openRequest: Bool,
    NFTProviderPath: PrivatePath
) {
    prepare(pds: AuthAccount) {
        let cap = pds.borrow<&PDS.DistributionManager>(from: PDS.DistManagerStoragePath) ?? panic("pds does not have Dist manager")
        let p = {{.PackNFTName}}.borrowPackRepresentation(id: packId) ?? panic ("No such pack")
        if openRequest && p.status == {{.PackNFTName}}.Status.Revealed {
            let recvAcct = getAccount(owner)
            let recv = recvAcct.getCapability({{.CollectibleNFTName}}.CollectionPublicPath).borrow<&{NonFungibleToken.CollectionPublic}>()
                ?? panic("Unable to borrow Collection Public reference for recipient")
            cap.openPackNFT(
                distId: distId,
                packId: packId,
                nftContractAddrs: nftContractAddrs,
                nftContractName: nftContractName,
                nftIds: nftIds,
                recvCap: recv,
                collectionProviderPath: NFTProviderPath
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
