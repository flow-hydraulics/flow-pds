import NonFungibleToken from 0x{{.NonFungibleToken}}
import PackNFT from 0x{{.PackNFT}}

transaction(recipient: Address, withdrawID: UInt64) {
    prepare(signer: AuthAccount) {
        let recipient = getAccount(recipient)

        // borrow a reference to the signer's NFT collection
        let collectionRef = signer
            .borrow<&PackNFT.Collection>(from: PackNFT.CollectionStoragePath)!

        // borrow a public reference to the receivers collection
        let depositRef = recipient
            .getCapability(PackNFT.CollectionPublicPath)!
            .borrow<&{NonFungibleToken.CollectionPublic}>()!

        // withdraw the NFT from the owner's collection
        let nft <- collectionRef.withdraw(withdrawID: withdrawID)

        // Deposit the NFT in the recipient's collection
        depositRef.deposit(token: <-nft)
    }
}
