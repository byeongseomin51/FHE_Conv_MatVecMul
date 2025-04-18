package core

import (
	"strconv"

	"github.com/tuneinsight/lattigo/v5/core/rlwe"
	"github.com/tuneinsight/lattigo/v5/schemes/ckks"
)

// ///////////////////////////////////////////////////////////////////////////////////////////////////////
// Implementation of Rotation Optimized DownSampling.
// ///////////////////////////////////////////////////////////////////////////////////////////////////////
type RotOptDS struct {
	Evaluator     *ckks.Evaluator
	preCompStride *rlwe.Plaintext
	preCompFilter []*rlwe.Plaintext
	params        ckks.Parameters
	planes        int
	rotateNums    []int
}

func NewRotOptDS(planes int, ev *ckks.Evaluator, ec *ckks.Encoder, params ckks.Parameters) *RotOptDS {

	//declare
	preCompFilters := make([]*rlwe.Plaintext, 4)

	//make plaintext
	path := "mulParModules/precomputed/rotOptDS/" + strconv.Itoa(planes) + "/"
	preCompStrideFilter := txtToPlain(ec, path+"stride.txt", params)
	for i := 0; i < 4; i++ {
		preCompFilters[i] = txtToPlain(ec, path+"filter"+strconv.Itoa(i)+".txt", params)
	}

	rotateNums := []int{}
	if planes == 16 {
		rotateNums = []int{
			1024 - 1, 2048 - 32,
			-2048, 1024, 4096, 1024 * 7,
			8192,
		}

	} else if planes == 32 {
		rotateNums = []int{
			32 - 2, 2048 - 64,
			-1024, -32, 2048, 1024*3 - 32,
			4096,
		}

	}
	return &RotOptDS{
		Evaluator:     ev,
		preCompStride: preCompStrideFilter,
		preCompFilter: preCompFilters,
		params:        params,
		planes:        planes,
		rotateNums:    rotateNums,
	}
}
func (obj RotOptDS) Foward(ctIn *rlwe.Ciphertext) (ctOut *rlwe.Ciphertext) {

	tempCipher, _ := obj.Evaluator.MulRelinNew(ctIn, obj.preCompStride)
	obj.Evaluator.Rescale(tempCipher, tempCipher)

	tempCipher2 := ckks.NewCiphertext(obj.params, ctIn.Degree(), ctIn.Level())

	for i := 0; i < 2; i++ {
		obj.Evaluator.Rotate(tempCipher, obj.rotateNums[i], tempCipher2)
		obj.Evaluator.Add(tempCipher, tempCipher2, tempCipher)
	}

	for i := 0; i < 4; i++ {
		if i == 0 {
			tempCipher2, err := obj.Evaluator.MulRelinNew(tempCipher, obj.preCompFilter[i])
			ErrorPrint(err)
			obj.Evaluator.Rescale(tempCipher2, tempCipher2)
			ctOut, _ = obj.Evaluator.RotateNew(tempCipher2, obj.rotateNums[i+2])
		} else {
			tempCipher2, err := obj.Evaluator.MulRelinNew(tempCipher, obj.preCompFilter[i])
			ErrorPrint(err)
			obj.Evaluator.Rescale(tempCipher2, tempCipher2)
			obj.Evaluator.Rotate(tempCipher2, obj.rotateNums[i+2], tempCipher2)
			obj.Evaluator.Add(ctOut, tempCipher2, ctOut)
		}
	}

	obj.Evaluator.Rotate(ctOut, obj.rotateNums[6], tempCipher)
	obj.Evaluator.Add(ctOut, tempCipher, ctOut)

	return ctOut
}

func RotOptDSRegister() []int {

	rotateNums := []int{
		//planes==16
		1024 - 1, 2048 - 32,
		-2048, 1024, 4096, 1024 * 7,
		8192,

		//planes==32
		32 - 2, 2048 - 64,
		-1024, -32, 2048, 1024*3 - 32,
		4096,
	}

	return rotateNums

}

func MakeTxtRotOptDS() {
	path := "mulParModules/precomputed/rotOptDS/"

	// declare stride filter
	stride16 := make([]float64, 32768)
	stride32 := make([]float64, 32768)

	// make stride filter
	k := 2
	for index := 0; index < 32768; index++ {
		if index%k < (k/2) && ((index/32)%k < (k / 2)) {
			stride16[index] = 1
		}
	}

	k = 4
	for index := 0; index < 32768; index++ {
		if index%k < (k/2) && ((index/32)%k < (k / 2)) {
			stride32[index] = 1
		}
	}

	// save stride filter
	floatToTxt(path+"16/stride.txt", stride16)
	floatToTxt(path+"32/stride.txt", stride32)

	// declare valid filter
	filter16 := make([][]float64, 4)
	for i := range filter16 {
		filter16[i] = make([]float64, 32768)
	}

	filter32 := make([][]float64, 4)
	for i := range filter32 {
		filter32[i] = make([]float64, 32768)
	}

	// make stride filter
	for i := 0; i < 4; i++ {
		for index := 0; index < 32768; index++ {
			if ((index % 16384) >= 4096*i) && ((index % 16384) < 4096*i+1024) {
				filter16[i][index] = 1
			}
		}
	}

	for i := 0; i < 4; i++ {
		for index := 0; index < 32768; index++ {
			start := (i%2)*1024 + (i/2)*4096
			if ((index % 8192) >= start) && ((index % 8192) < start+1024) && (((index%8192)/32)%2 == 0) {
				filter32[i][index] = 1
			}
		}
	}

	// save valid filter
	for i := 0; i < 4; i++ {
		floatToTxt(path+"16/filter"+strconv.Itoa(i)+".txt", filter16[i])
	}

	for i := 0; i < 4; i++ {
		floatToTxt(path+"32/filter"+strconv.Itoa(i)+".txt", filter32[i])
	}

}
