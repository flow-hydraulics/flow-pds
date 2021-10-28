import PDS from 0x{{.PDS}}
import NonFungibleToken from 0x{{.NonFungibleToken}}

transaction (distId: UInt64, state: UInt8) {
    // state is an enum
    // - 0: Initialized
    // - 1: Invalid
    // - 2: Complete
    prepare(pds: AuthAccount) {
        let cap = pds.borrow<&PDS.DistributionManager>(from: PDS.DistManagerStoragePath) ?? panic("pds does not have Dist manager")
        cap.updateDistState(
            distId: distId,
            state: PDS.DistState(rawValue: state)!,
        )
    }
}
