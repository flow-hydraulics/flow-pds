## Goals:

- Create a "blind" Pack Distribution Service (PDS) on the Flow blockchain.
- Create a smart contract or set of smart contracts that facilitate interaction with the PDS

Note: "blind" is used in this document to mean “the contents of which are hidden from public view”

## NFT Types and Containers

### Collectible

> An NFT intended to be distributed in blind packs (e.g. Top Shot Moments).

- MUST be NFTs as defined by the `NonFungibleToken` core contract. As such, they MUST be
  transferrable, storable, and sellable like any other NFT on the Flow blockchain.

### Pack

> An NFT that represents a claim to some predefined set of Collectibles.

- MUST be NFTs as defined by the `NonFungibleToken` core contract. As such, they MUST be
  transferrable, storable, and sellable like any other NFT on the Flow blockchain.
  - SHOULD BE _compatible_ with other project's NFTs by e.g. the pack distribution capability of
    NBA Top Shot, however MUST be secured such that such compatibility MUST NOT be realized.
- MUST NOT contain a random salt value before being revealed
- MUST contain a random salt value after the Pack has been revealed for verifying the _commitment hash_
- MUST contain a _commitment hash_ of the contents of the Pack plus the salt value
- MUST maintain static content, i.e. MUST NOT change their salt value or _commitment hash_ over time
- MUST report a _state_: one of `"Building"`, `"Sealed"`, `"Revealed"`, `"Empty"`, or `"Invalid"`
- MUST have content expectations validated by the PDS
  - MAY use the post-condition state of the Flow transaction to further validate
- MUST contain a `Reveal` function which:
  - sets the `Pack` NFT state as `"Revealed"`
  - emits an on-chain event to be observed by the PDS with the `Pack` ID
- MUST contain a `Withdraw` function which:
  - withdraws ALL Collectible NFTs from the `Pack` NFT


### Distributions

> A server-side PDS construct that represents the configuration and contents of Pack NFTs

- MUST contain the (public) set of Pack NFTs
- MUST NOT reveal the set of Collectible NFTs within Pack NFTs
- SHOULD keep the revelation of unopened Pack NFT contents (by process of elimination) to a minimum
- MUST report a "state": one of "editable", "complete" or "invalid"
- MUST not allow for re-use of a distribution in a "complete" state

## Roles

### PDS

The back-end service that the smart contract will interact with to create Distributions

- SHOULD only be accessibly via be the Cadence smart contract(s)
  - MAY be accessible via an API for the first iteration(s)
- MUST allow Issuers to configure Collectible "Buckets" for managing rarity tiers
- MUST assign Collectibles to Packs according to the Issuer's configuration
  - Collectibles MAY be assigned at the time of pack assignment to reduce transactions
- MUST allow Issuers to configure a _Pack Template_ to assign NFTs to buckets.
  - **Example 1**: A simple Pack of three Collectibles, all pulled from a single Bucket
  - **Example 2**: A Pack of four Collectibles, the first slot is filled by pulling from Bucket
    one (“uncommons”), the second from a mix of Buckets one and two (Bucket two is “rares”)
    and slots three and four come from Bucket three (“commons”)
- MUST assign Collectible NFTs to Packs off-chain
- MUST asynchronously determine if the configuration of Pack Template, Collectibles, Buckets, and number of Packs is valid
  - MUST emit an off-chain response reporting any such invalid configurations to the Issuer
- MUST provide a long-term storage container for Collectible NFTs after creation Pack creation
  - MUST only approve withdrawal during the pack opening process
  - MUST split withdrawals randomly into transactions to obfuscate information about pack contents
- MUST observe "reveal" events emitted by the PackReceiverCap with a Pack NFT ID (see below)
  - MUST respond with an on-chain event with the IDs of the Collectible NFTs within the Pack NFT
  - MUST check that the pack content hash (plus a salt value) matches the hash of the pack

### Issuer

> The entity that is requesting the creation and distribution capability of Packs from the PDS

- MUST be represented by a Flow account
- MUST have a sharable Cadence Capability for the creation of Pack NFT resources.
- MUST be capable of transferring all required Collectible NFTs into the PDS
- MAY or MAY NOT be the minter of the Collectibles
- MAY or MAY NOT be the administrator of the PDS instance

### Owner

> The entity that comes into possession of a Pack NFT

- MUST be represented by a Flow account
- MUST be able to receive the `PackReceiverCap` (a normal NFT `Receiver` object for the `Pack` NFT type) capability
- MUST be able to `Reveal` the contents of an NFT pack
- MUST be able to `Withdraw` Collectible NFTs from a successfully "Revealed" pack

## Capabilities

### DistCap

> Distribution capability

- MUST be configured by the Issuer via the PDS before the capability is finally granted
- MUST allow the issuer to transfer Pack NFTs.
- MAY abort distribution at any time before a successful creation event
- MUST provide an emergency abort function that can be triggered in certain circumstances
  e.g. the PDS malfunctions
- SHOULD BE discarded after the creation period

### PackReceiverCap

> NFT Pack Receiver Capability

- MUST receive Pack NFTs after successful creation by the PDS
