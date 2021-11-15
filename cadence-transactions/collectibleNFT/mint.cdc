import NonFungibleToken from 0x{{.NonFungibleToken}}
import {{.CollectibleNFTName}} from 0x{{.CollectibleNFTAddress}}

// Used for testing purposes

transaction(recipient: Address, batchSize: Int) {

    let minter: &{{.CollectibleNFTName}}.NFTMinter

    prepare(signer: AuthAccount) {
        self.minter = signer
            .borrow<&{{.CollectibleNFTName}}.NFTMinter>(from: {{.CollectibleNFTName}}.MinterStoragePath)!
    }

    execute {
        let receiver = getAccount(recipient)
            .getCapability({{.CollectibleNFTName}}.CollectionPublicPath)!
            .borrow<&{NonFungibleToken.CollectionPublic}>()!

        var i = 0
        while i < batchSize {
            self.minter.mintNFT(recipient: receiver)
            i = i + 1
        }
    }
}
