import NonFungibleToken from 0x{{.NonFungibleToken}}
import {{.CollectibleNFTName}} from 0x{{.CollectibleNFTAddress}}

pub fun main(account: Address): Int {
    let receiver = getAccount(account)
        .getCapability({{.CollectibleNFTName}}.CollectionPublicPath)!
        .borrow<&{NonFungibleToken.CollectionPublic}>()!

    return receiver.getIDs().length
}
