import PackNFT from 0x{{.PackNFT}}
import IPackNFT from 0x{{.IPackNFT}}

pub fun main(account: Address, id: UInt64): String {
    let receiver = getAccount(account)
        .getCapability(PackNFT.collectionIPackNFTPublicPath)!
        .borrow<&{IPackNFT.IPackNFTCollection}>()!

    let nft = receiver.borrowPackNFT(id: id) 
    return nft.commitHash
}
