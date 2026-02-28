package dilithium

import (
	"bytes"
	"testing"
)

func TestGenerateKeyPair(t *testing.T) {
	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	if pub == nil || priv == nil {
		t.Fatal("nil key returned")
	}
}

func TestSignVerify(t *testing.T) {
	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}

	msg := []byte("hello probechain dilithium")
	sig := Sign(priv, msg)

	if len(sig) != SignatureSize {
		t.Fatalf("signature size: got %d, want %d", len(sig), SignatureSize)
	}

	if !Verify(pub, msg, sig) {
		t.Error("valid signature rejected")
	}

	// Tamper with message
	if Verify(pub, []byte("tampered"), sig) {
		t.Error("tampered message accepted")
	}

	// Tamper with signature
	badSig := make([]byte, len(sig))
	copy(badSig, sig)
	badSig[0] ^= 0xff
	if Verify(pub, msg, badSig) {
		t.Error("tampered signature accepted")
	}
}

func TestMarshalRoundtrip(t *testing.T) {
	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}

	// Marshal/unmarshal private key
	privBytes := MarshalPrivateKey(priv)
	if len(privBytes) != PrivateKeySize {
		t.Fatalf("private key size: got %d, want %d", len(privBytes), PrivateKeySize)
	}
	priv2, err := UnmarshalPrivateKey(privBytes)
	if err != nil {
		t.Fatalf("UnmarshalPrivateKey: %v", err)
	}
	privBytes2 := MarshalPrivateKey(priv2)
	if !bytes.Equal(privBytes, privBytes2) {
		t.Error("private key roundtrip failed")
	}

	// Marshal/unmarshal public key
	pubBytes := MarshalPublicKey(pub)
	if len(pubBytes) != PublicKeySize {
		t.Fatalf("public key size: got %d, want %d", len(pubBytes), PublicKeySize)
	}
	pub2, err := UnmarshalPublicKey(pubBytes)
	if err != nil {
		t.Fatalf("UnmarshalPublicKey: %v", err)
	}
	pubBytes2 := MarshalPublicKey(pub2)
	if !bytes.Equal(pubBytes, pubBytes2) {
		t.Error("public key roundtrip failed")
	}

	// Verify signature with roundtripped keys
	msg := []byte("roundtrip test")
	sig := Sign(priv2, msg)
	if !Verify(pub2, msg, sig) {
		t.Error("signature verification failed with roundtripped keys")
	}
}

func TestPubkeyToAddress(t *testing.T) {
	pub, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}

	addr := PubkeyToAddress(pub)
	// Address should be 20 bytes and non-zero (extremely unlikely to be all zeros)
	allZero := true
	for _, b := range addr {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("address is all zeros")
	}

	// Same pubkey should produce same address
	addr2 := PubkeyToAddress(pub)
	if addr != addr2 {
		t.Error("deterministic address derivation failed")
	}
}

func TestPublicFromPrivate(t *testing.T) {
	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}

	derivedPub := priv.Public()
	if derivedPub == nil {
		t.Fatal("Public() returned nil")
	}

	// Public keys should match
	pubBytes := MarshalPublicKey(pub)
	derivedBytes := MarshalPublicKey(derivedPub)
	if !bytes.Equal(pubBytes, derivedBytes) {
		t.Error("derived public key doesn't match original")
	}
}

func TestInvalidKeySize(t *testing.T) {
	_, err := UnmarshalPrivateKey([]byte{1, 2, 3})
	if err == nil {
		t.Error("expected error for invalid private key size")
	}

	_, err = UnmarshalPublicKey([]byte{1, 2, 3})
	if err == nil {
		t.Error("expected error for invalid public key size")
	}
}
