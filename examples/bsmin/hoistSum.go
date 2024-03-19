package main

import (
	"fmt"
	"math"
	"time"

	"github.com/tuneinsight/lattigo/v5/core/rlwe"
	"github.com/tuneinsight/lattigo/v5/schemes/ckks"
)

func GetOptSumNum(ctLevel int, c_l int) (int, int) {
	if ctLevel == 2 && c_l == 8 { //experiment
		return 8, 1
	}

	//c_l =2
	if c_l == 2 {
		return 2, 1
	}
	if ctLevel <= 5 {
		return c_l, 1
	} else if ctLevel > 5 && c_l == 4 {
		return 1, 4
	} else if ctLevel > 5 && c_l == 8 {
		return 2, 4
	} else if ctLevel > 6 && c_l == 16 {
		return 1, 16
	} else {
		fmt.Println("Error in GetOptSumNum(). Doesn't have OptSumNum data for ctLevel : ", ctLevel, " and c_l : ", c_l)
		return c_l, 1
	}
}

// Rot Index example : {1024,2048,4096} for combine 1024,2048,3072,4096,5120,6144,7168
func OptHoistSum(ctIn *rlwe.Ciphertext, rotIndex []int, evaluator *ckks.Evaluator) *rlwe.Ciphertext {
	startTime := time.Now()

	ctOut := ctIn.CopyNew()
	ctLevel := ctIn.Level()
	c_l := math.Pow(2, float64(len(rotIndex)))

	GetOptSumNum(ctLevel, int(c_l))
	n_c, _ := GetOptSumNum(ctLevel, int(c_l)) //HSumNum

	currentRotIndexLocate := 0

	endTime := time.Now()
	fmt.Println(endTime.Sub(startTime))

	startTime = time.Now()
	//For n_c
	for o := 1; o < n_c; o *= 2 {
		tempCt, _ := evaluator.RotateNew(ctOut, rotIndex[currentRotIndexLocate])
		fmt.Println(rotIndex[currentRotIndexLocate])
		evaluator.Add(ctOut, tempCt, ctOut)
		currentRotIndexLocate++
	}

	endTime = time.Now()
	fmt.Println(endTime.Sub(startTime))
	startTime = time.Now()

	//여기서 currentRotIndexLocate = log_2{n_c} 임!!!
	// if math.Pow(2, float64(currentRotIndexLocate)) == float64(n_c) {
	// 	fmt.Println("??", math.Pow(2, float64(currentRotIndexLocate)), float64(n_c))
	// }
	//Make New Rot Index
	//newRotIndexLen = c_l/n_c
	newRotIndexLen := int(math.Pow(2, float64(len(rotIndex)-currentRotIndexLocate))) - 1
	newRotIndex := make([]int, newRotIndexLen)
	for bit := 1; bit <= newRotIndexLen; bit++ {
		temp := 0
		for index := currentRotIndexLocate; index < len(rotIndex); index++ {
			fmt.Println(rotIndex[index], ((bit >> (index - currentRotIndexLocate)) & 1))
			if ((bit >> (index - currentRotIndexLocate)) & 1) == 1 {
				temp += rotIndex[index]
			}
		}
		fmt.Println("temp", temp)
		newRotIndex[bit-1] = temp
	}
	endTime = time.Now()
	fmt.Println(endTime.Sub(startTime))

	startTime = time.Now()
	// Make New Rot Index
	fmt.Println(newRotIndex)
	ctOuts, _ := evaluator.RotateHoistedNew(ctOut, newRotIndex)
	for _, c := range ctOuts {
		evaluator.Add(ctOut, c, ctOut)
	}

	endTime = time.Now()
	fmt.Println(endTime.Sub(startTime))

	// startTime := time.Now()
	// ctOuts, _ := evaluator.RotateHoistedNew(ctOut, rotIndex)
	// for c := range ctOuts {
	// 	evaluator.Add(ctOut, c, ctOut)
	// }
	// endTime := time.Now()
	// // fmt.Println("hoist!", rotIndex, endTime.Sub(startTime))

	// startTime = time.Now()
	// for i := 1; i <= len(rotIndex); i *= 2 {
	// 	// fmt.Println(rotIndex[i-1], "??")
	// 	temp, _ := evaluator.RotateNew(ctOut, rotIndex[i-1])
	// 	evaluator.Add(temp, ctOut, ctOut)
	// }
	// endTime = time.Now()
	// endTime.Sub(startTime)
	// // fmt.Println("conventional!", endTime.Sub(startTime))
	return ctOut
}

func FindOptHoist(decompose float64, other float64, channelLen int) (timeResult float64, conventionalLen int, hoistLen int) {
	if channelLen == 1 {
		return 0, 1, 1
	}
	if channelLen == 2 {
		return decompose + other, 2, 1
	}

	// If using previous values
	minTime := 1000000.0
	resultC := 1
	resultH := 1
	for i := 2; i < channelLen; i *= 2 {
		time1, o1, h1 := FindOptHoist(decompose, other, i)
		time2, o2, h2 := FindOptHoist(decompose, other, channelLen/i)
		if minTime > time1+time2 {
			minTime = time1 + time2
			resultC = o1 * o2
			resultH = h1 * h2
		}
	}

	//If using only conventional
	onlyC := 0.0
	for i := 1; i < channelLen; i *= 2 {
		onlyC += decompose + other
	}
	if minTime > onlyC {
		minTime = onlyC
		resultC = channelLen
		resultH = 1
	}

	//If using only hoist
	onlyH := decompose
	for i := 1; i < channelLen; i++ {
		onlyH += other
	}
	if minTime > onlyH {
		minTime = onlyH
		resultC = 1
		resultH = channelLen
	}

	return minTime, resultC, resultH

}

func min(a, b float64) float64 {
	if a < b {
		return a
	} else {
		return b
	}
}
