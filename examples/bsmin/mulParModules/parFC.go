package mulParModules

import (
	"strconv"

	"github.com/tuneinsight/lattigo/v5/core/rlwe"
	"github.com/tuneinsight/lattigo/v5/schemes/ckks"
)

type ParFC struct {
	Evaluator     *ckks.Evaluator
	preCompBias   *rlwe.Plaintext
	preCompWeight []*rlwe.Plaintext
	layerNum      int
	params        ckks.Parameters
}

func NewparFC(ev *ckks.Evaluator, ec *ckks.Encoder, params ckks.Parameters, layer int) *ParFC {
	// fmt.Println("FC : ", layer)
	path := "mulParModules/precomputed/parFC/" + strconv.Itoa(layer) + "/"

	//declare
	preCompWeights := []*rlwe.Plaintext{}
	//make plaintext

	for i := 0; i < 10; i++ {
		preCompWeights = append(preCompWeights, txtToPlain(ec, path+"weight"+strconv.Itoa(i)+".txt", params))
	}

	return &ParFC{
		Evaluator:     ev,
		preCompBias:   txtToPlain(ec, path+"bias.txt", params),
		preCompWeight: preCompWeights,
		layerNum:      layer,
		params:        params,
	}
}
func (obj ParFC) Foward(ctIn *rlwe.Ciphertext) (ctOut *rlwe.Ciphertext) {

	ct_rot_index := []int{-8, -7, -6, -5, -4, -3, -2, -1, 0}
	result_rot_index := []int{4096 + 9, 8192 + 18, 16384 + 36}

	//Make initializer
	tempRelin, _ := obj.Evaluator.RotateNew(ctIn, -9)
	ctOut, err := obj.Evaluator.MulNew(tempRelin, obj.preCompWeight[9])
	ErrorPrint(err)

	tempCipher := ckks.NewCiphertext(obj.params, ctIn.Degree()+obj.preCompWeight[0].Degree(), ctIn.Level())
	for i := 0; i < 9; i++ {
		if ct_rot_index[i] == 0 {
			relinTemp, err := obj.Evaluator.MulRelinNew(ctIn, obj.preCompWeight[i])
			ErrorPrint(err)
			obj.Evaluator.Add(ctOut, relinTemp, ctOut)
		} else {
			obj.Evaluator.Rotate(ctIn, ct_rot_index[i], tempCipher)
			relinTemp, err := obj.Evaluator.MulRelinNew(tempCipher, obj.preCompWeight[i])
			ErrorPrint(err)
			obj.Evaluator.Add(ctOut, relinTemp, ctOut)
		}
	}

	obj.Evaluator.Rescale(ctOut, ctOut)

	for i := 0; i < 3; i++ {
		obj.Evaluator.Rotate(ctOut, result_rot_index[i], tempCipher)
		obj.Evaluator.Add(ctOut, tempCipher, ctOut)
	}

	obj.Evaluator.Add(ctOut, obj.preCompBias, ctOut)
	return ctOut
}

func ParFCRegister() []int {

	rotateNums := []int{-9, -8, -7, -6, -5, -4, -3, -2, -1,
		4096 + 9, 8192 + 18, 16384 + 36,
	}

	return rotateNums

}
