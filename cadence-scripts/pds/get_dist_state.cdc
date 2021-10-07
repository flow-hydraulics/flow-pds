import PDS from 0x{{.PDS}}

pub fun main(distId: UInt64): UInt8 {
    return PDS.getDistInfo(distId: distId)!.state.rawValue
}
