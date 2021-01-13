// Copyright 2020 ConsenSys Software Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package eddsa

import (
	"crypto/sha256"
	"testing"

	"github.com/consensys/gnark/crypto/hash/mimc/bn256"
	"github.com/consensys/gurvy/bn256/fr"
)

func TestSerialization(t *testing.T) {

	var seed [32]byte
	s := []byte("eddsa")
	for i, v := range s {
		seed[i] = v
	}

	pubKey, privKey := New(seed)
	hFunc := sha256.New()
	signature, err := Sign([]byte("message"), privKey, hFunc)
	if err != nil {
		t.Fatal("unexpected error when signing")
	}

	marshalPubKey := pubKey.Marshal()
	marshalprivKey := privKey.Marshal()
	marshalSignature := signature.Marshal()

	var unMarshalPubKey PublicKey
	var unMarshalPrivKey PrivateKey
	var unMarshalSignature Signature

	unMarshalPubKey.Unmarshal(marshalPubKey)
	unMarshalPrivKey.Unmarshal(marshalprivKey)
	unMarshalSignature.Unmarshal(marshalSignature)

	// public key
	if !unMarshalPubKey.A.Equal(&pubKey.A) {
		t.Fatal("unmarshal(marshal(pubkey)) failed")
	}

	// signature
	if !unMarshalSignature.R.Equal(&signature.R) {
		t.Fatal("unmarshal(marshal(signature.R)) failed")
	}
	for i := 0; i < frSize; i++ {
		if unMarshalSignature.S[i] != signature.S[i] {
			t.Fatal("unmarshal(marshal(signature.S)) failed")
		}
	}

	// private key
	if !privKey.pubKey.A.Equal(&unMarshalPrivKey.pubKey.A) {
		t.Fatal("unmarshal(marshal(privKey.pubkey)) failed")
	}
	for i := 0; i < 32; i++ {
		if privKey.randSrc[i] != unMarshalPrivKey.randSrc[i] {
			t.Fatal("unmarshal(marshal(privKey.randSrc)) failed")
		}
	}
	for i := 0; i < frSize; i++ {
		if privKey.scalar[i] != unMarshalPrivKey.scalar[i] {
			t.Fatal("unmarshal(marshal(signature.scalar)) failed")
		}
	}

}

func TestEddsaMIMC(t *testing.T) {

	var seed [32]byte
	s := []byte("eddsa")
	for i, v := range s {
		seed[i] = v
	}

	hFunc := bn256.NewMiMC("seed")

	// create eddsa obj and sign a message
	pubKey, privKey := New(seed)
	var frMsg fr.Element
	frMsg.SetString("44717650746155748460101257525078853138837311576962212923649547644148297035978")
	msgBin := frMsg.Bytes()
	signature, err := Sign(msgBin[:], privKey, hFunc)
	if err != nil {
		t.Fatal(err)
	}

	// verifies correct msg
	res, err := Verify(signature, msgBin[:], pubKey, hFunc)
	if err != nil {
		t.Fatal(err)
	}
	if !res {
		t.Fatal("Verifiy correct signature should return true")
	}

	// verifies wrong msg
	frMsg.SetString("44717650746155748460101257525078853138837311576962212923649547644148297035979")
	msgBin = frMsg.Bytes()
	res, err = Verify(signature, msgBin[:], pubKey, hFunc)
	if err != nil {
		t.Fatal(err)
	}
	if res {
		t.Fatal("Verfiy wrong signature should be false")
	}

}

func TestEddsaSHA256(t *testing.T) {

	var seed [32]byte
	s := []byte("eddsa")
	for i, v := range s {
		seed[i] = v
	}

	hFunc := sha256.New()

	// create eddsa obj and sign a message
	pubKey, privKey := New(seed)
	signature, err := Sign([]byte("message"), privKey, hFunc)
	if err != nil {
		t.Fatal(err)
	}

	// verifies correct msg
	res, err := Verify(signature, []byte("message"), pubKey, hFunc)
	if err != nil {
		t.Fatal(err)
	}
	if !res {
		t.Fatal("Verifiy correct signature should return true")
	}

	// verifies wrong msg
	res, err = Verify(signature, []byte("wrong_message"), pubKey, hFunc)
	if err != nil {
		t.Fatal(err)
	}
	if res {
		t.Fatal("Verfiy wrong signature should be false")
	}

}

// benchmarks

func BenchmarkVerify(b *testing.B) {

	var seed [32]byte
	s := []byte("eddsa")
	for i, v := range s {
		seed[i] = v
	}

	hFunc := bn256.NewMiMC("seed")

	// create eddsa obj and sign a message
	pubKey, privKey := New(seed)
	var frMsg fr.Element
	frMsg.SetString("44717650746155748460101257525078853138837311576962212923649547644148297035978")
	msgBin := frMsg.Bytes()
	signature, _ := Sign(msgBin[:], privKey, hFunc)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Verify(signature, msgBin[:], pubKey, hFunc)
	}
}
