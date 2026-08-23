package domain

import "testing"

func TestCanonicalChain_CMCLabels(t *testing.T) {
	if CanonicalChain("BNB Smart Chain (BEP20)") != "bsc" {
		t.Fatal(CanonicalChain("BNB Smart Chain (BEP20)"))
	}
	if CanonicalChain("Optimism") != "optimism" {
		t.Fatal(CanonicalChain("Optimism"))
	}
	if CanonicalChain("optimistic-ethereum") != "optimism" {
		t.Fatal(CanonicalChain("optimistic-ethereum"))
	}
	if CanonicalChain("Base") != "base" {
		t.Fatal(CanonicalChain("Base"))
	}
	if CanonicalChain("tron20") != "tron" {
		t.Fatal(CanonicalChain("tron20"))
	}
	if CanonicalChain("zkSync Era") != "zksync" {
		t.Fatal(CanonicalChain("zkSync Era"))
	}
}

func TestInferContractChain(t *testing.T) {
	if InferContractChain("Solana", "8WbN") != "solana" {
		t.Fatal("keep explicit chain")
	}
	if InferContractChain("", "0x02300475d1EdD5b2E88EFdeBD3fFb549110D8Aa6") != "ethereum" {
		t.Fatal("0x should infer ethereum")
	}
	if InferContractChain("tron20", "TCFLL5dx5ZJdKnWuesXxi1VPwjLVmWZZy9") != "tron" {
		t.Fatal(InferContractChain("tron20", "TCFLL5dx5ZJdKnWuesXxi1VPwjLVmWZZy9"))
	}
	if InferContractChain("", "TCFLL5dx5ZJdKnWuesXxi1VPwjLVmWZZy9") != "tron" {
		t.Fatal("tron address inferred as", InferContractChain("", "TCFLL5dx5ZJdKnWuesXxi1VPwjLVmWZZy9"))
	}
	if InferContractChain("", "8WbNQtY7QmXMVKJFTSqFudierVZZtbuoyeepZEqJ1B2w") != "solana" {
		t.Fatal("base58 should infer solana")
	}
}
