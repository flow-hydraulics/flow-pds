import Crypto
pub fun main(toHash: String): String {
    let hashB2 = HashAlgorithm.SHA2_256.hash(toHash.utf8)
    log(String.encodeHex(hashB2))
    return String.encodeHex(hashB2)
}
