import PackNFT from 0x{{.PackNFT}}
import IPackNFT from 0x{{.IPackNFT}}

pub fun main(id: UInt64): UInt8 {
    let p = PackNFT.borrowPackRepresentation(id: id) 
    return p!.status.rawValue
}
