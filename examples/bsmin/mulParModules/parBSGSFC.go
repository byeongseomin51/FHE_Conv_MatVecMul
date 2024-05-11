package mulParModules

import (
	"math"
	"strconv"

	"github.com/tuneinsight/lattigo/v5/core/rlwe"
	"github.com/tuneinsight/lattigo/v5/schemes/ckks"
)

type ParBSGSFC struct {
	Evaluator        *ckks.Evaluator
	preCompBias      *rlwe.Plaintext
	preCompWeight    []*rlwe.Plaintext
	layerNum         int
	params           ckks.Parameters
	n2               int
	n1               int
	pi               int
	ct_rot_index     []int
	result_rot_index []int
}

func NewParBSGSFC(ev *ckks.Evaluator, ec *ckks.Encoder, params ckks.Parameters, layer int) *ParBSGSFC {
	n2 := 32
	n1 := 2
	pi := 16

	// fmt.Println("ParBSGSFC : ", layer)
	path := "mulParModules/precomputed/resnetPtParam/" + strconv.Itoa(layer) + "/"

	//declare
	A := make([][]float64, 64)
	for j := 0; j < 64; j++ {
		A[j] = make([]float64, 64)
	}
	txtA := txtToFloat(path + "linear_weight.txt")
	//make plaintext
	for i := 0; i < 10; i++ {
		for j := 0; j < 64; j++ {
			A[i][j] = txtA[i*64+j]
		}
	}

	//diagonalized
	d := make([][]float64, 64)
	for i := 0; i < 64; i++ {
		d[i] = make([]float64, 32768)
		for j := 0; j < 64; j++ {
			index := j + i
			if index >= 64 {
				index = index % 64
			}
			d[i][j] = A[j][index]
		}
	}

	//Make precompweights
	D := make([][]float64, n1)
	for i := 0; i < n1; i++ {
		D[i] = make([]float64, 32768)
		for j := 0; j < n2; j++ {
			iD := rotate(d[j*n1+i], -n1*j-(32768/n2)*j)
			D[i] = add(D[i], iD)
		}
	}

	preCompWeight := make([]*rlwe.Plaintext, n1)
	for i := 0; i < n1; i++ {
		preCompWeight[i] = floatToPlain(D[i], ec, params)
	}

	//make rotation indexes
	ct_rot_index := make([]int, n1)
	for i := 0; i < n1; i++ {
		ct_rot_index[i] = i
	}
	result_rot_index := make([]int, 0)
	for i := 1; i < n2; i *= 2 {
		result_rot_index = append(result_rot_index, n1*i+(32768/n2)*i)
	}

	return &ParBSGSFC{
		Evaluator:        ev,
		preCompBias:      txtToPlain(ec, path+"linear_bias.txt", params),
		preCompWeight:    preCompWeight,
		layerNum:         layer,
		params:           params,
		n1:               n1,
		n2:               n2,
		pi:               pi,
		ct_rot_index:     ct_rot_index,
		result_rot_index: result_rot_index,
	}
}
func (obj ParBSGSFC) Foward(ctIn *rlwe.Ciphertext) (ctOut *rlwe.Ciphertext) {
	initial_rot := []int{-64, -1024, -2048}

	//copy
	for _, i := range initial_rot {
		// fmt.Println(i)
		c, err := obj.Evaluator.RotateNew(ctIn, i)
		ErrorPrint(err)
		err = obj.Evaluator.Add(ctIn, c, ctIn)
		ErrorPrint(err)
	}

	ctOut, err := obj.Evaluator.MulNew(ctIn, obj.preCompWeight[0])
	ErrorPrint(err)
	for i := 1; i < obj.n1; i++ {
		rotatedIn, err := obj.Evaluator.RotateNew(ctIn, obj.ct_rot_index[i])
		ErrorPrint(err)
		tempB, err := obj.Evaluator.MulNew(rotatedIn, obj.preCompWeight[i])
		ErrorPrint(err)
		err = obj.Evaluator.Add(ctOut, tempB, ctOut)
		ErrorPrint(err)

	}
	err = obj.Evaluator.Rescale(ctOut, ctOut)
	ErrorPrint(err)

	for i := 0; i < int(math.Log2(float64(obj.n2))); i++ {
		tempAnswer, err := obj.Evaluator.RotateNew(ctOut, obj.result_rot_index[i])
		ErrorPrint(err)
		err = obj.Evaluator.Add(ctOut, tempAnswer, ctOut)
		ErrorPrint(err)
	}

	err = obj.Evaluator.Add(ctOut, obj.preCompBias, ctOut)
	ErrorPrint(err)

	return ctOut
}

func ParBSGSFCRegister() []int {
	result := make([]int, 0)
	n2 := 32
	n1 := 2

	for i := 0; i < n1; i++ {
		result = append(result, i)
	}

	for i := 1; i < n2; i *= 2 {
		result = append(result, n1*i+(32768/n2)*i)
	}
	result = append(result, -64)
	result = append(result, -2048)
	result = append(result, -1024)

	return result

}
