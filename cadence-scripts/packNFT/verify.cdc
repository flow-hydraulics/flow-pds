import PackNFT from 0x{{.PackNFT}}
import IPackNFT from 0x{{.IPackNFT}}

pub fun main(id: UInt64, nftString: String): Bool {
    let p = PackNFT.borrowPackRepresentation(id: id) 
    return p!.verify(nftString: nftString)
}
