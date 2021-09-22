import PackNFT from 0x{{.PackNFT}}
import IPackNFT from 0x{{.IPackNFT}}

transaction(revealID: UInt64) {
    prepare(owner: AuthAccount) {
        // withdraw the token from collection
        let collectionRef = owner.borrow<&PackNFT.Collection>(from: PackNFT.collectionStoragePath)!
        let nft <- collectionRef.withdraw(withdrawID: revealID) as! @PackNFT.NFT
        // reveal
        nft.reveal()

        // store token back to collection
        collectionRef.deposit(token: <-nft) 
    }
}