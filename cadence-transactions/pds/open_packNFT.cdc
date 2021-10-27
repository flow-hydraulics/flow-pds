import PDS from 0x{{.PDS}}
import ExampleNFT from 0x{{.ExampleNFT}}
import NonFungibleToken from 0x{{.NonFungibleToken}}

transaction (distId: UInt64, packId: UInt64, nftContractAddrs: [Address], nftContractName: [String], nftIds: [UInt64], owner: Address, NFTProviderPath: PrivatePath) {
    prepare(pds: AuthAccount) {
        let cap = pds.borrow<&PDS.DistributionManager>(from: PDS.DistManagerStoragePath) ?? panic("pds does not have Dist manager")
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
            collectionProviderPath: NFTProviderPath, 
        )
    }
}

