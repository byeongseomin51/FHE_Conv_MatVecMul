package mulParModules

import (
	"fmt"
	"strconv"

	"github.com/tuneinsight/lattigo/v5/core/rlwe"
	"github.com/tuneinsight/lattigo/v5/schemes/ckks"
)

type AvgPool struct {
	Evaluator    *ckks.Evaluator
	preCompPlain []*rlwe.Plaintext
	params       ckks.Parameters
}

func NewAvgPool(ev *ckks.Evaluator, ec *ckks.Encoder, params ckks.Parameters) *AvgPool {
	fmt.Println("AvgPool")
	//declare
	preCompPlaintext := []*rlwe.Plaintext{}
	//make plaintext
	path := "mulParModules/precomputed/avgPool/filter"
	for i := 0; i < 16; i++ {
		preCompPlaintext = append(preCompPlaintext, txtToPlain(ec, path+strconv.Itoa(i)+".txt", params))
	}

	return &AvgPool{
		Evaluator:    ev,
		preCompPlain: preCompPlaintext,
		params:       params,
	}
}
func (this AvgPool) Foward(ctIn *rlwe.Ciphertext) (ctOut *rlwe.Ciphertext) {
	var err error

	rotateNums := []int{4, 8, 16, 128, 256, 512}
	linearizeNums := []int{
		0, 1008, 2016, 3024,
		28, 1036, 2044, 3052,
		56, 1064, 2072, 3080,
		84, 1092, 2100, 3108,
	}

	tempCipher := ckks.NewCiphertext(this.params, ctIn.Degree(), ctIn.Level())
	for i := 0; i < 6; i++ {
		this.Evaluator.Rotate(ctIn, rotateNums[i], tempCipher)
		this.Evaluator.Add(tempCipher, ctIn, ctIn)
	}

	for i := 0; i < 16; i++ {
		if i == 0 {
			ctOut, err = this.Evaluator.MulRelinNew(ctIn, this.preCompPlain[i])
			ErrorPrint(err)
			this.Evaluator.Rescale(ctOut, ctOut)
		} else {
			tempRelin, err := this.Evaluator.MulRelinNew(ctIn, this.preCompPlain[i])
			ErrorPrint(err)
			this.Evaluator.Rescale(tempRelin, tempCipher)
			this.Evaluator.Rotate(tempCipher, linearizeNums[i], tempCipher)
			this.Evaluator.Add(ctOut, tempCipher, ctOut)
		}
	}
	return ctOut
}

func AvgPoolRegister() []int {

	rotateNums := []int{4, 8, 16, 128, 256, 512,
		1008, 2016, 3024,
		28, 1036, 2044, 3052,
		56, 1064, 2072, 3080,
		84, 1092, 2100, 3108,
	}

	return rotateNums

}
