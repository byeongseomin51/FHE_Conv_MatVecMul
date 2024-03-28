package mulParModules

import (
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
	// fmt.Println("AvgPool")
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
func (obj AvgPool) Foward(ctIn *rlwe.Ciphertext) (ctOut *rlwe.Ciphertext) {
	var err error

	rotateNums := []int{4, 8, 16, 128, 256, 512}
	linearizeNums := []int{
		0, 1008, 2016, 3024,
		28, 1036, 2044, 3052,
		56, 1064, 2072, 3080,
		84, 1092, 2100, 3108,
	}

	tempCipher := ckks.NewCiphertext(obj.params, ctIn.Degree(), ctIn.Level())
	for i := 0; i < 6; i++ {
		obj.Evaluator.Rotate(ctIn, rotateNums[i], tempCipher)
		obj.Evaluator.Add(tempCipher, ctIn, ctIn)
	}

	for i := 0; i < 16; i++ {
		if i == 0 {
			ctOut, err = obj.Evaluator.MulRelinNew(ctIn, obj.preCompPlain[i])
			ErrorPrint(err)
			obj.Evaluator.Rescale(ctOut, ctOut)
		} else {
			tempRelin, err := obj.Evaluator.MulRelinNew(ctIn, obj.preCompPlain[i])
			ErrorPrint(err)
			obj.Evaluator.Rescale(tempRelin, tempCipher)
			obj.Evaluator.Rotate(tempCipher, linearizeNums[i], tempCipher)
			obj.Evaluator.Add(ctOut, tempCipher, ctOut)
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
