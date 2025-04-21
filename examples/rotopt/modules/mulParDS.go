package modules

import (
	"fmt"

	"github.com/tuneinsight/lattigo/v5/core/rlwe"
	"github.com/tuneinsight/lattigo/v5/schemes/ckks"
)

// ///////////////////////////////////////////////////////////////////////////////////////////////////////
// Conventional multiplexed parallel downsampling
// ///////////////////////////////////////////////////////////////////////////////////////////////////////
type MulParDS struct {
	Evaluator      *ckks.Evaluator
	preCompFilters []*rlwe.Plaintext
	params         ckks.Parameters
	planes         int
	rotChannel     []int
	rotCopy        []int
}

func NewMulParDS(planes int, ev *ckks.Evaluator, ec *ckks.Encoder, params ckks.Parameters) *MulParDS {

	//declare
	preCompFilters := make([]*rlwe.Plaintext, 16)
	rotChannel := make([]int, 16)
	//make plaintext
	if planes == 16 {
		for channel := 0; channel < 16; channel++ {
			preCompFilters[channel] = floatToPlain(multVec(StrideFilter(1, 32), GeneralFilter((channel%8)*4, channel/8, 2)), ec, params)

			originalLocate := 1024 * channel
			resultLocate := 2048 + (channel/4)*1024 + (channel%4)/2*32 + channel%4%2
			rotChannel[channel] = originalLocate - resultLocate
		}
	} else if planes == 32 { // 0 4 16 20 32 36 48 52
		for channel := 0; channel < 16; channel++ {
			f := AndVec(GeneralFilter((channel%8)%2*4+(channel%8)/2*16, channel/8, 4), GeneralFilter((channel%8)%2*4+(channel%8)/2*16+1, channel/8, 4))
			preCompFilters[channel] = floatToPlain(multVec(StrideFilter(2, 16), f), ec, params)
			originalLocate := 1024*(channel/2) + 32*(channel%2)
			resultLocate := 1024 + (channel/8)*1024 + (channel%8)/2*32 + (channel%8%2)*2
			rotChannel[channel] = originalLocate - resultLocate
		}
	}

	//For copy
	rotCopy := make([]int, 0)

	if planes == 16 {
		rotCopy = []int{-8192, -16384}

	} else if planes == 32 {
		rotCopy = []int{-4096, -8192, -16384}
	}

	// fmt.Println(rotChannel)
	// fmt.Println(rotCopy)
	return &MulParDS{
		Evaluator:      ev,
		preCompFilters: preCompFilters,
		params:         params,
		planes:         planes,
		rotChannel:     rotChannel,
		rotCopy:        rotCopy,
	}
}
func (obj MulParDS) Foward(ctIn *rlwe.Ciphertext) (ctOut *rlwe.Ciphertext) {
	for i := 0; i < 16; i++ {
		temp, err := obj.Evaluator.MulNew(ctIn, obj.preCompFilters[i])
		ErrorPrint(err)
		obj.Evaluator.Rescale(temp, temp)

		if i == 0 {
			ctOut, err = obj.Evaluator.RotateNew(temp, obj.rotChannel[i])
			ErrorPrint(err)
		} else {
			temp2, err := obj.Evaluator.RotateNew(temp, obj.rotChannel[i])
			ErrorPrint(err)
			obj.Evaluator.Add(ctOut, temp2, ctOut)
		}

	}

	if obj.planes == 16 {
		for i := 0; i < 2; i++ {
			temp, err := obj.Evaluator.RotateNew(ctOut, obj.rotCopy[i])
			ErrorPrint(err)
			err = obj.Evaluator.Add(ctOut, temp, ctOut)
			ErrorPrint(err)
		}
	}
	if obj.planes == 32 {
		for i := 0; i < 3; i++ {
			temp, err := obj.Evaluator.RotateNew(ctOut, obj.rotCopy[i])
			ErrorPrint(err)
			err = obj.Evaluator.Add(ctOut, temp, ctOut)
			ErrorPrint(err)
		}
	}
	return ctOut
}

func MulParDSRegister() []int {

	rotateNums := []int{-2048, -1025, -32, 991, 1024, 2047, 3040, 4063, 4096, 5119, 6112, 7135, 7168, 8191, 9184, 10207,
		-8192, -16384,
		-1024, -994, -32, -2, 960, 990, 1952, 1982, 2048, 2078, 3040, 3070, 4032, 4062, 5024, 5054,
		-4096, -8192, -16384}

	return rotateNums

}

// count continuous zero and non-zero
func Count01num(arr []float64) {
	currentZero := true
	if zeroFilter(arr[0]) != 0 {
		currentZero = false
	}
	count := 0
	for i := 0; i < 32768; i++ {
		cur := zeroFilter(arr[i])
		if currentZero && (cur == 0.0) {
			count++
		} else if (currentZero == false) && (cur != 0.0) {
			count++
		} else {
			if currentZero {
				currentZero = false
				fmt.Printf("0 : %v\n", count)
				count = 1
			} else {
				currentZero = true
				fmt.Printf("1 : %v\n", count)
				count = 1
			}
		}
	}
	if currentZero {
		currentZero = false
		fmt.Printf("0 : %v\n", count)
		count = 0
	} else {
		currentZero = true
		fmt.Printf("1 : %v\n", count)
		count = 0
	}

}
func zeroFilter(input float64) float64 {
	if input > 0.00001 || input < -0.00001 {
		return input
	} else {
		return 0
	}
}
