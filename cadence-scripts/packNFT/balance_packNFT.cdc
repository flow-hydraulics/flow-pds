import NonFungibleToken from 0x{{.NonFungibleToken}}
import PackNFT from 0x{{.PackNFT}}

pub fun main(account: Address): [UInt64] {
    let receiver = getAccount(account)
        .getCapability(PackNFT.CollectionPublicPath)!
        .borrow<&{NonFungibleToken.CollectionPublic}>()!

    return receiver.getIDs()
}
