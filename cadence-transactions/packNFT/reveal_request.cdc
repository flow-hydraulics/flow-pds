import PackNFT from 0x{{.PackNFT}}
import IPackNFT from 0x{{.IPackNFT}}

transaction(revealID: UInt64) {
    prepare(owner: AuthAccount) {
        let collectionRef = owner.borrow<&PackNFT.Collection>(from: PackNFT.collectionStoragePath)!
        collectionRef.borrowPackNFT(id: revealID)!.reveal()
    }
}
