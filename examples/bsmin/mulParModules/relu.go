package mulParModules

import (
	"github.com/tuneinsight/lattigo/v5/core/rlwe"
	"github.com/tuneinsight/lattigo/v5/schemes/ckks"
)

type Relu struct {
	Evaluator *ckks.Evaluator
	Encoder   *ckks.Encoder
	Decryptor *rlwe.Decryptor
	Encryptor *rlwe.Encryptor
	params    ckks.Parameters
}

func NewRelu(Evaluator *ckks.Evaluator, Encoder *ckks.Encoder, Decryptor *rlwe.Decryptor, Encryptor *rlwe.Encryptor, params ckks.Parameters) *Relu {

	return &Relu{
		Evaluator: Evaluator,
		Encoder:   Encoder,
		Decryptor: Decryptor,
		Encryptor: Encryptor,
		params:    params,
	}
}
func (this Relu) Foward(ctIn *rlwe.Ciphertext) (ctOut *rlwe.Ciphertext) {
	// fmt.Println("Relu: ")

	//Temp Relu
	inputPlain := ckks.NewPlaintext(this.params, this.params.MaxLevel())
	inputFloat := make([]float64, this.params.MaxSlots())
	this.Decryptor.Decrypt(ctIn, inputPlain)
	this.Encoder.Decode(inputPlain, inputFloat)

	for i := 0; i < len(inputFloat); i++ {
		if inputFloat[i] < 0 {
			inputFloat[i] = 0
		}
	}

	outputPlain := ckks.NewPlaintext(this.params, 5)
	this.Encoder.Encode(inputFloat, outputPlain)
	ctOut, _ = this.Encryptor.EncryptNew(outputPlain)

	return ctOut
}
