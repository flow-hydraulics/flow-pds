transaction(idx: Int) {
    prepare(signer: AuthAccount) {
      let keyA = signer.keys.revoke(keyIndex: idx)
    }
}
