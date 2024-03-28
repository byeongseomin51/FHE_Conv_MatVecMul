package mulParModules

import (
	"strconv"

	"github.com/tuneinsight/lattigo/v5/core/rlwe"
	"github.com/tuneinsight/lattigo/v5/schemes/ckks"
)

type MulParDS struct {
	Evaluator     *ckks.Evaluator
	preCompStride *rlwe.Plaintext
	preCompFilter []*rlwe.Plaintext
	params        ckks.Parameters
	planes        int
	rotateNums    []int
}

func NewMulParDS(planes int, ev *ckks.Evaluator, ec *ckks.Encoder, params ckks.Parameters) *MulParDS {

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
			1024 - 1, 2048 - 32, //filter 전
			-2048, 1024, 4096, 1024 * 7, //filter 후
			8192, // 복사
		}

	} else if planes == 32 {
		rotateNums = []int{
			32 - 2, 2048 - 64, //filter 전
			-1024, -32, 2048, 1024*3 - 32, //filter 후
			4096, //복사
		}

	}
	return &MulParDS{
		Evaluator:     ev,
		preCompStride: preCompStrideFilter,
		preCompFilter: preCompFilters,
		params:        params,
		planes:        planes,
		rotateNums:    rotateNums,
	}
}
func (obj MulParDS) Foward(ctIn *rlwe.Ciphertext) (ctOut *rlwe.Ciphertext) {

	forNum := 0
	if obj.planes == 16 {
		forNum = 18
	} else if obj.planes == 32 {
		forNum = 19
	}

	for i := 0; i < forNum; i++ {
		temp, _ := obj.Evaluator.MulNew(ctIn, obj.preCompStride)
		obj.Evaluator.Rescale(temp, temp)
		if i == 0 {
			ctOut, _ = obj.Evaluator.RotateNew(temp, 4096)
		} else {
			temp2, _ := obj.Evaluator.RotateNew(temp, 4096)
			obj.Evaluator.Add(ctOut, temp2, ctOut)
		}
	}
	return ctOut
}

func MulParDSRegister() []int {

	rotateNums := []int{
		//planes==16
		1024 - 1, 2048 - 32, //filter 전
		-2048, 1024, 4096, 1024 * 7, //filter 후
		8192, // 복사

		//planes==32
		32 - 2, 2048 - 64, //filter 전
		-1024, -32, 2048, 1024*3 - 32, //filter 후
		4096, //복사
	}

	return rotateNums

}

func MakeTxtMulParDS() {
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
