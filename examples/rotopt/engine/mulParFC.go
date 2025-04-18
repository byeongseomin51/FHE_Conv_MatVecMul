package engine

import (
	"strconv"

	"github.com/tuneinsight/lattigo/v5/core/rlwe"
	"github.com/tuneinsight/lattigo/v5/schemes/ckks"
)

// ///////////////////////////////////////////////////////////////////////////////////////////////////////
// Fully Connected Layer of ResNet20(CIFAR-10) implementation using conventional BSGS diagonal method.
// ///////////////////////////////////////////////////////////////////////////////////////////////////////
type MulParFC struct {
	Evaluator       *ckks.Evaluator
	preCompLeftover *rlwe.Plaintext
	preCompBias     *rlwe.Plaintext
	preCompWeight   [][]*rlwe.Plaintext
	layerNum        int
	params          ckks.Parameters
}

func NewMulParFC(ev *ckks.Evaluator, ec *ckks.Encoder, params ckks.Parameters, layer int) *MulParFC {
	// fmt.Println("MulParFC : ", layer)
	path := "engine/precomputed/parFC/" + strconv.Itoa(layer) + "/"

	//declare
	weights := make([][]float64, 10)
	//make plaintext
	for i := 0; i < 10; i++ {
		weights[i] = txtToFloat(path + "weight" + strconv.Itoa(i) + ".txt")
	}

	//Make precompweights
	preCompWeights := make([][]*rlwe.Plaintext, 8)
	for par := 0; par < 8; par++ {
		preCompWeights[par] = make([]*rlwe.Plaintext, 9)
		for i := 0; i < 9; i++ {
			tempFilter := rotate(crossFilter(weights[i], 8)[par], 4096*par)
			preCompWeights[par][i] = floatToPlain(tempFilter, ec, params)
		}
	}

	return &MulParFC{
		Evaluator:       ev,
		preCompLeftover: floatToPlain(weights[9], ec, params),
		preCompBias:     txtToPlain(ec, path+"bias.txt", params),
		preCompWeight:   preCompWeights,
		layerNum:        layer,
		params:          params,
	}
}
func (obj MulParFC) Foward(ctIn *rlwe.Ciphertext) (ctOut *rlwe.Ciphertext) {

	ct_rot_index := []int{-8, -7, -6, -5, -4, -3, -2, -1, 0}
	result_rot_index := []int{0, 9, 18, 27, 36, 45, 54, 63}

	var rotInput []*rlwe.Ciphertext
	for i := 0; i < 9; i++ {
		c, err := obj.Evaluator.RotateNew(ctIn, ct_rot_index[i])
		ErrorPrint(err)
		rotInput = append(rotInput, c)
	}

	//Make initializer
	ctOut, _ = obj.Evaluator.RotateNew(ctIn, -9)
	relinTemp, err := obj.Evaluator.MulRelinNew(ctOut, obj.preCompLeftover)
	ErrorPrint(err)
	obj.Evaluator.Rescale(relinTemp, ctOut)

	for par := 0; par < 8; par++ {
		tempCipher := ckks.NewCiphertext(obj.params, 1, ctIn.Level())
		for i := 0; i < 9; i++ {
			if i == 0 {
				tempCipher, err = obj.Evaluator.MulNew(rotInput[i], obj.preCompWeight[par][i])
				ErrorPrint(err)
			} else {
				relinTemp, err := obj.Evaluator.MulNew(rotInput[i], obj.preCompWeight[par][i])
				ErrorPrint(err)
				err = obj.Evaluator.Add(relinTemp, tempCipher, tempCipher)
				ErrorPrint(err)
			}
		}
		err = obj.Evaluator.Rescale(tempCipher, tempCipher)
		ErrorPrint(err)
		temp2, err := obj.Evaluator.RotateNew(tempCipher, result_rot_index[par])
		ErrorPrint(err)
		obj.Evaluator.Add(ctOut, temp2, ctOut)
	}
	obj.Evaluator.Add(ctOut, obj.preCompBias, ctOut)
	return ctOut
}

func MulParFCRegister() []int {

	rotateNums := []int{-8, -7, -6, -5, -4, -3, -2, -1,
		9, 18, 27, 36, 45, 54, 63,
	}

	return rotateNums

}
