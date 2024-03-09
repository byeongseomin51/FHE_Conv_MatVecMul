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
func (this ParFC) Foward(ctIn *rlwe.Ciphertext) (ctOut *rlwe.Ciphertext) {

	ct_rot_index := []int{-8, -7, -6, -5, -4, -3, -2, -1, 0}
	result_rot_index := []int{4096 + 9, 8192 + 18, 16384 + 36}

	//Make initializer
	ctOut, _ = this.Evaluator.RotateNew(ctIn, -9)
	relinTemp, err := this.Evaluator.MulRelinNew(ctOut, this.preCompWeight[9])
	ErrorPrint(err)
	this.Evaluator.Rescale(relinTemp, ctOut)

	tempCipher := ckks.NewCiphertext(this.params, ctIn.Degree()+this.preCompWeight[0].Degree(), ctIn.Level())
	for i := 0; i < 9; i++ {
		if ct_rot_index[i] == 0 {
			relinTemp, err := this.Evaluator.MulRelinNew(ctIn, this.preCompWeight[i])
			ErrorPrint(err)
			this.Evaluator.Rescale(relinTemp, relinTemp)
			this.Evaluator.Add(ctOut, relinTemp, ctOut)
		} else {
			this.Evaluator.Rotate(ctIn, ct_rot_index[i], tempCipher)
			relinTemp, err := this.Evaluator.MulRelinNew(tempCipher, this.preCompWeight[i])
			ErrorPrint(err)
			this.Evaluator.Rescale(relinTemp, relinTemp)
			this.Evaluator.Add(ctOut, relinTemp, ctOut)
		}
	}

	for i := 0; i < 3; i++ {
		this.Evaluator.Rotate(ctOut, result_rot_index[i], tempCipher)
		this.Evaluator.Add(ctOut, tempCipher, ctOut)
	}

	this.Evaluator.Add(ctOut, this.preCompBias, ctOut)
	return ctOut
}

func ParFCRegister() []int {

	rotateNums := []int{-9, -8, -7, -6, -5, -4, -3, -2, -1,
		4096 + 9, 8192 + 18, 16384 + 36,
	}

	return rotateNums

}
