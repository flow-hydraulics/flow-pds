import NonFungibleToken from 0x{{.NonFungibleToken}}
import {{.CollectibleNFTName}} from 0x{{.CollectibleNFTAddress}}

pub fun main(account: Address, offset: UInt64, limit: UInt64): [UInt64] {
    let receiver = getAccount(account)
        .getCapability({{.CollectibleNFTName}}.CollectionPublicPath)!
        .borrow<&{NonFungibleToken.CollectionPublic}>()!

    let ids = receiver.getIDs()
    let idsLen = UInt64(ids.length)

    var res: [UInt64] = []
    var i = offset
    while i < offset+limit && i < idsLen {
        res.append(ids[i])
        i = i + 1
    }

    return res
}
