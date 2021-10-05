import PDS from 0x{{.PDS}}

pub fun main(distId: UInt64): {String: String} {
    return PDS.getDistInfo(distId: distId)!.metadata
}
