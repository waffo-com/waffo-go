package vectors

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/waffo-com/waffo-go/v2/utils"
)

type rsaVector struct {
	Module            string                `json:"module"`
	Algorithm         rsaAlgorithm          `json:"algorithm"`
	TestKeys          rsaTestKeys           `json:"testKeys"`
	SigningTests      []rsaSigningTest      `json:"signingTests"`
	VerificationTests []rsaVerificationTest `json:"verificationTests"`
}

type rsaTestKeys struct {
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
}

type rsaAlgorithm struct {
	Name             string `json:"name"`
	StandardName     string `json:"standardName"`
	KeySize          int    `json:"keySize"`
	PrivateKeyFormat string `json:"privateKeyFormat"`
	PublicKeyFormat  string `json:"publicKeyFormat"`
}

type rsaSigningTest struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Input struct {
		Data string `json:"data"`
	} `json:"input"`
	Output struct {
		Signature string `json:"signature"`
	} `json:"output"`
}

type rsaVerificationTest struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Input struct {
		Data                string `json:"data"`
		Signature           string `json:"signature"`
		TamperedData        string `json:"tamperedData"`
		SignatureOfOriginal string `json:"signatureOfOriginal"`
	} `json:"input"`
	Expected struct {
		Valid bool `json:"valid"`
	} `json:"expected"`
}

// loadRSAVector reads the registered monorepo vector used by every RSA suite.
func loadRSAVector(t *testing.T) rsaVector {
	t.Helper()
	data, err := os.ReadFile("../../sdk-spec/test-vectors/rsa-signing.json")
	if err != nil {
		t.Fatalf("read rsa-signing.json: %v", err)
	}

	var vector rsaVector
	if err := json.Unmarshal(data, &vector); err != nil {
		t.Fatalf("parse rsa-signing.json: %v", err)
	}
	if vector.Module != "rsa-signing" {
		t.Fatalf("unexpected RSA vector module: %q", vector.Module)
	}
	return vector
}

func TestRSAAlgorithmVector(t *testing.T) {
	algorithm := loadRSAVector(t).Algorithm
	if algorithm.Name != "SHA256withRSA" || algorithm.StandardName != "RSASSA-PKCS1-v1_5" {
		t.Fatalf("unsupported RSA algorithm vector: %+v", algorithm)
	}
	if algorithm.KeySize != 2048 {
		t.Fatalf("unexpected RSA key size: %d", algorithm.KeySize)
	}
	if algorithm.PrivateKeyFormat == "" || algorithm.PublicKeyFormat == "" {
		t.Fatal("RSA vector must declare both key formats")
	}
}

func TestRSASigningVectors(t *testing.T) {
	vector := loadRSAVector(t)

	for _, testCase := range vector.SigningTests {
		t.Run(testCase.ID+"_"+testCase.Name, func(t *testing.T) {
			signature, err := utils.Sign(testCase.Input.Data, vector.TestKeys.PrivateKey)
			if err != nil {
				t.Fatalf("Sign() failed: %v", err)
			}
			if signature != testCase.Output.Signature {
				t.Fatalf("Sign() did not match canonical signature for %s", testCase.ID)
			}
			if !utils.Verify(testCase.Input.Data, signature, vector.TestKeys.PublicKey) {
				t.Fatal("Verify() rejected the canonical signature")
			}
		})
	}
}

func TestRSAVerificationVectors(t *testing.T) {
	vector := loadRSAVector(t)

	for _, testCase := range vector.VerificationTests {
		t.Run(testCase.ID+"_"+testCase.Name, func(t *testing.T) {
			data := testCase.Input.Data
			signature := testCase.Input.Signature
			if testCase.Input.TamperedData != "" {
				data = testCase.Input.TamperedData
				signature = testCase.Input.SignatureOfOriginal
			}
			actual := utils.Verify(data, signature, vector.TestKeys.PublicKey)

			if actual != testCase.Expected.Valid {
				t.Fatalf("Verify()=%v, expected %v", actual, testCase.Expected.Valid)
			}
		})
	}
}
