import NonFungibleToken from "./NonFungibleToken.cdc"

pub contract interface PDSInterface{
    access(contract) let Distributions: {UInt64: CapabilityPath}
    /// Issuer has created a distribution 
    ///
    pub event DistributionCreated(DistId: UInt64)
}
