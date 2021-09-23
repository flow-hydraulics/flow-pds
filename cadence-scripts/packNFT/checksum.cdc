import Crypto
pub fun main(): String {
    let toHash = "f24dfdf9911df152,A.01cf0e2f2f715450.ExampleNFT.0,A.01cf0e2f2f715450.ExampleNFT.3"
    let hashB2 = HashAlgorithm.SHA2_256.hash(toHash.utf8)
    log(String.encodeHex(hashB2))
    return String.encodeHex(hashB2)
}