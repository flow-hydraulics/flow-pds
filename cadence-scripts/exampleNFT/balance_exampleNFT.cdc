import NonFungibleToken from 0x{{.NonFungibleToken}}
import ExampleNFT from 0x{{.ExampleNFT}}

pub fun main(account: Address): [UInt64] {
    let receiver = getAccount(account)
        .getCapability(ExampleNFT.CollectionPublicPath)!
        .borrow<&{NonFungibleToken.CollectionPublic}>()!

    return receiver.getIDs()
}
