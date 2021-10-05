import PDS from 0x{{.PDS}}
import NonFungibleToken from 0x{{.NonFungibleToken}}

transaction (distId: UInt64, state: String) {
    prepare(pds: AuthAccount) {
        let cap = pds.borrow<&PDS.DistributionManager>(from: PDS.distManagerStoragePath) ?? panic("pds does not have Dist manager")
        cap.updateDistState(
            distId: distId,
            state: state, 
        )
    }
}

