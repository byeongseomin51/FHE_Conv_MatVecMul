package modules

import (
	"fmt"

	"github.com/tuneinsight/lattigo/v5/core/rlwe"
	"github.com/tuneinsight/lattigo/v5/schemes/ckks"
)

// ///////////////////////////////////////////////////////////////////////////////////////////////////////
// Implementation of Rotation Optimized Convolution.
// ///////////////////////////////////////////////////////////////////////////////////////////////////////
type RotOptConv struct {
	encoder *ckks.Encoder

	Evaluator            *ckks.Evaluator
	params               ckks.Parameters
	PreCompKernels       [][]*rlwe.Plaintext
	preCompFilters       [][]*rlwe.Plaintext
	lastFilter           [][]*rlwe.Plaintext
	lastFilterTreeDepth  int
	opType0TreeDepth     int
	opType1LastTreeDepth int
	ConvFeature          *ConvFeature

	convMap            [][]int
	q                  int //length of kernel_map
	rotIndex3by3Kernel []int

	splitNum int
	depth    int

	//for debug
	rot_num int
}

func NewrotOptConv(ev *ckks.Evaluator, ec *ckks.Encoder, params ckks.Parameters, convID string, depth int) *RotOptConv {
	//rotOptConv Setting
	convMap, q, rotIndex3by3Kernel := GetConvBlueprints(convID, depth)

	// conv feature
	cf := GetRotOptConvFeature(convID)

	// plaintext setting, kernel weight
	var preCompFilters [][]*rlwe.Plaintext
	var lastFilter [][]*rlwe.Plaintext

	// preCompFilters, lastFilter generate
	preCompFilters, lastFilter = MakeTxtRotOptConvFilter(convID, depth, ec, params)

	// get splitNum, lastFilterLocate, opType0TreeDepth value
	splitNum := 0
	lastFilterLocate := 0
	opType0TreeDepth := 0 //last opType0 or opType2 treeDepth locate
	for depth := len(convMap) - 1; depth > 0; depth-- {
		opType := convMap[depth][0]
		if opType == 2 {
			lastFilterLocate = depth
			opType0TreeDepth = depth
		} else if opType == 3 {
			splitNum = convMap[depth][1]
		} else if opType == 0 {
			opType0TreeDepth = depth
		} else {

		}
	}

	//get opType1CombineNum values.
	var opType1LastTreeDepth = 0
	for depth := opType0TreeDepth; depth > 0; depth-- {
		opType := convMap[depth][0]
		if opType == 1 {
			opType1LastTreeDepth = depth
			break
		}
	}

	return &RotOptConv{
		encoder: ec,

		Evaluator:            ev,
		params:               params,
		preCompFilters:       preCompFilters,
		lastFilter:           lastFilter,
		lastFilterTreeDepth:  lastFilterLocate,
		opType0TreeDepth:     opType0TreeDepth,
		opType1LastTreeDepth: opType1LastTreeDepth,
		ConvFeature:          cf,
		splitNum:             splitNum,

		convMap:            convMap,
		q:                  q,
		rotIndex3by3Kernel: rotIndex3by3Kernel,
		depth:              depth,

		//debug
		rot_num: -1,
	}
}

func (obj *RotOptConv) Foward2depth(ctIn *rlwe.Ciphertext) (ctOut *rlwe.Ciphertext) {
	mainCipher := ckks.NewCiphertext(obj.params, 1, ctIn.Level())
	tempCtLv1 := ckks.NewCiphertext(obj.params, 1, ctIn.Level())

	var err error
	// Rotate Data
	var rotInput []*rlwe.Ciphertext
	for w := 0; w < 9; w++ {
		c, err := obj.Evaluator.RotateNew(ctIn, obj.rotIndex3by3Kernel[w])
		// obj.rot_num++
		ErrorPrint(err)
		rotInput = append(rotInput, c)
	}

	// conv
	var splitedCiphertext []*rlwe.Ciphertext
	for i := 0; i < obj.ConvFeature.q; i++ {
		kernelResult, err := obj.Evaluator.MulNew(rotInput[0], obj.PreCompKernels[i][0])
		ErrorPrint(err)
		for w := 1; w < 9; w++ {
			tempCt, err := obj.Evaluator.MulNew(rotInput[w], obj.PreCompKernels[i][w])
			ErrorPrint(err)
			err = obj.Evaluator.Add(kernelResult, tempCt, kernelResult)
			ErrorPrint(err)
		}
		err = obj.Evaluator.Rescale(kernelResult, mainCipher)
		ErrorPrint(err)

		//opType 0
		for treeDepth := obj.opType0TreeDepth; treeDepth < obj.lastFilterTreeDepth; treeDepth++ {
			err = obj.Evaluator.Rotate(mainCipher, obj.convMap[treeDepth][1], tempCtLv1)
			// obj.rot_num++
			ErrorPrint(err)
			err = obj.Evaluator.Add(mainCipher, tempCtLv1, mainCipher)
			ErrorPrint(err)
		}

		// opType2, rotate
		shift := 0
		for j := 1; j < obj.convMap[obj.lastFilterTreeDepth][1]; j *= 2 {
			if ((i >> shift) & 1) == 0 {
				err = obj.Evaluator.Rotate(mainCipher, obj.convMap[obj.lastFilterTreeDepth][shift+2], tempCtLv1)
				// obj.rot_num++
				ErrorPrint(err)
				err = obj.Evaluator.Add(mainCipher, tempCtLv1, mainCipher)
				ErrorPrint(err)
			} else {
				err = obj.Evaluator.Rotate(mainCipher, -obj.convMap[obj.lastFilterTreeDepth][shift+2], tempCtLv1)
				// obj.rot_num++
				ErrorPrint(err)
				err = obj.Evaluator.Add(mainCipher, tempCtLv1, mainCipher)
				ErrorPrint(err)
			}
			shift++
		}

		//opType2, split
		for s := 0; s < obj.splitNum; s++ {
			if i == 0 {
				tempCt, err := obj.Evaluator.MulNew(mainCipher, obj.lastFilter[i][s])
				ErrorPrint(err)
				splitedCiphertext = append(splitedCiphertext, tempCt)
			} else {
				tempCt, err := obj.Evaluator.MulNew(mainCipher, obj.lastFilter[i][s])
				ErrorPrint(err)
				err = obj.Evaluator.Add(splitedCiphertext[s], tempCt, splitedCiphertext[s])
				ErrorPrint(err)

			}
		}

	}
	//Rescaling
	for s := 0; s < obj.splitNum; s++ {
		err = obj.Evaluator.Rescale(splitedCiphertext[s], splitedCiphertext[s])
		ErrorPrint(err)
	}

	for i := 1; i < obj.convMap[obj.lastFilterTreeDepth+1][1]; i++ {
		err = obj.Evaluator.Rotate(splitedCiphertext[i], obj.convMap[obj.lastFilterTreeDepth+1][i+1], splitedCiphertext[i])
		// obj.rot_num++
		ErrorPrint(err)
		err = obj.Evaluator.Add(splitedCiphertext[0], splitedCiphertext[i], splitedCiphertext[0])
		ErrorPrint(err)
	}

	//copy paste
	for treeDepth := obj.lastFilterTreeDepth + 2; treeDepth < len(obj.convMap); treeDepth++ {
		if obj.convMap[treeDepth][0] == 0 {
			err = obj.Evaluator.Rotate(splitedCiphertext[0], obj.convMap[treeDepth][1], mainCipher)
			// obj.rot_num++
			ErrorPrint(err)
			err = obj.Evaluator.Add(splitedCiphertext[0], mainCipher, splitedCiphertext[0])
			ErrorPrint(err)
		} else {
			fmt.Println("Something wrong.. in RotOptConv.")
		}
	}

	// fmt.Println("rot num: ", obj.rot_num) //원
	return splitedCiphertext[0]
}

func (obj *RotOptConv) dac_for_opType1(ctOut *rlwe.Ciphertext, startKernel int, needsToBeCombine int, curTreeDepth int, rotInput []*rlwe.Ciphertext) {
	// if curTreeDepth ==0, return SISO conv
	if curTreeDepth == 0 {
		if needsToBeCombine != 1 {
			fmt.Println("Warning : curTreeDepth is 0, but needsToBeCombine is not 1")
		}
		kernelResult, err := obj.Evaluator.MulNew(rotInput[0], obj.PreCompKernels[startKernel][0])
		ErrorPrint(err)
		for w := 1; w < 9; w++ {
			tempCt, err := obj.Evaluator.MulNew(rotInput[w], obj.PreCompKernels[startKernel][w])
			ErrorPrint(err)

			err = obj.Evaluator.Add(kernelResult, tempCt, kernelResult)
			ErrorPrint(err)
		}

		err = obj.Evaluator.Rescale(kernelResult, ctOut)
		ErrorPrint(err)
	} else if curTreeDepth > 0 { // [startKernel, startKernel+needsToBeCombine) 까지 합쳐져야함
		ctOutTemp := ckks.NewCiphertext(obj.params, 1, ctOut.Level())

		// make constant
		curCombineNum := obj.convMap[curTreeDepth][1]
		curTreeLen := len(obj.convMap[curTreeDepth])

		//start divide and conquer
		curCtNum := 0
		obj.dac_for_opType1(ctOutTemp, startKernel, needsToBeCombine/curCombineNum, curTreeDepth-1, rotInput)
		tempCipher := ckks.NewCiphertext(obj.params, 1, ctOutTemp.Level())
		tempCipher2 := ckks.NewCiphertext(obj.params, 1, ctOutTemp.Level())
		ctOutTemp2 := ckks.NewCiphertext(obj.params, 1, ctOutTemp.Level())

		// rotate and add
		for index := 2; index < curTreeLen; index++ {
			if ((curCtNum >> (index - 2)) & 1) == 0 {
				err := obj.Evaluator.Rotate(ctOutTemp, obj.convMap[curTreeDepth][index], tempCipher)
				// obj.rot_num++
				ErrorPrint(err)
				err = obj.Evaluator.Add(ctOutTemp, tempCipher, ctOutTemp)
				ErrorPrint(err)
			} else {
				err := obj.Evaluator.Rotate(ctOutTemp, -obj.convMap[curTreeDepth][index], tempCipher)
				// obj.rot_num++
				ErrorPrint(err)
				err = obj.Evaluator.Add(ctOutTemp, tempCipher, ctOutTemp)
				ErrorPrint(err)
			}
		}
		// filter out
		err := obj.Evaluator.Mul(ctOutTemp, obj.preCompFilters[curTreeDepth][curCtNum], ctOutTemp2)
		ErrorPrint(err)

		for curCtNum = 1; curCtNum < curCombineNum; curCtNum++ {
			obj.dac_for_opType1(ctOutTemp, startKernel+curCtNum*(needsToBeCombine/curCombineNum), needsToBeCombine/curCombineNum, curTreeDepth-1, rotInput) //startKernel 수정
			// rotate and add
			for index := 2; index < curTreeLen; index++ {
				if ((curCtNum >> (index - 2)) & 1) == 0 {
					err := obj.Evaluator.Rotate(ctOutTemp, obj.convMap[curTreeDepth][index], tempCipher)
					// obj.rot_num++
					ErrorPrint(err)
					err = obj.Evaluator.Add(ctOutTemp, tempCipher, ctOutTemp)
					ErrorPrint(err)
				} else {
					err := obj.Evaluator.Rotate(ctOutTemp, -obj.convMap[curTreeDepth][index], tempCipher)
					// obj.rot_num++
					ErrorPrint(err)
					err = obj.Evaluator.Add(ctOutTemp, tempCipher, ctOutTemp)
					ErrorPrint(err)
				}
			}
			// filter out
			err := obj.Evaluator.Mul(ctOutTemp, obj.preCompFilters[curTreeDepth][curCtNum], tempCipher2)
			ErrorPrint(err)
			// add to result
			err = obj.Evaluator.Add(ctOutTemp2, tempCipher2, ctOutTemp2)
			ErrorPrint(err)
		}
		// rescale
		err = obj.Evaluator.Rescale(ctOutTemp2, ctOut)
		ErrorPrint(err)

	} else {
		fmt.Printf("curTreeDepth cannot be %d", curTreeDepth)
	}
}

func (obj *RotOptConv) Foward(ctIn *rlwe.Ciphertext) (ctOut *rlwe.Ciphertext) {
	obj.rot_num = -1 //원
	if obj.depth == 2 {
		return obj.Foward2depth(ctIn) //2 depth consuming rotation optimized convolution.
	}

	mainCipher := ckks.NewCiphertext(obj.params, 1, ctIn.Level())
	// tempCipher := ckks.NewCiphertext(obj.params, 1, ctIn.Level())
	tempCtLv1 := ckks.NewCiphertext(obj.params, 1, ctIn.Level())

	var err error
	// Rotate Data
	var rotInput []*rlwe.Ciphertext
	for w := 0; w < 9; w++ {
		c, err := obj.Evaluator.RotateNew(ctIn, obj.rotIndex3by3Kernel[w])
		// obj.rot_num++
		ErrorPrint(err)
		rotInput = append(rotInput, c)
	}

	// conv
	kernelNum := len(obj.ConvFeature.KernelBP)
	beforeLastFilter := kernelNum / obj.convMap[obj.lastFilterTreeDepth][1]
	var splitedCiphertext []*rlwe.Ciphertext
	for i := 0; i < obj.convMap[obj.lastFilterTreeDepth][1]; i++ {

		startKernel := beforeLastFilter * i
		obj.dac_for_opType1(mainCipher, startKernel, beforeLastFilter, obj.opType1LastTreeDepth, rotInput)

		//opType 0
		for treeDepth := obj.opType0TreeDepth; treeDepth < obj.lastFilterTreeDepth; treeDepth++ {
			err = obj.Evaluator.Rotate(mainCipher, obj.convMap[treeDepth][1], tempCtLv1)
			// obj.rot_num++
			ErrorPrint(err)
			err = obj.Evaluator.Add(mainCipher, tempCtLv1, mainCipher)
			ErrorPrint(err)
		}

		// opType2, rotate
		shift := 0
		for j := 1; j < obj.convMap[obj.lastFilterTreeDepth][1]; j *= 2 {
			if ((i >> shift) & 1) == 0 {
				err = obj.Evaluator.Rotate(mainCipher, obj.convMap[obj.lastFilterTreeDepth][shift+2], tempCtLv1)
				// obj.rot_num++
				ErrorPrint(err)
				err = obj.Evaluator.Add(mainCipher, tempCtLv1, mainCipher)
				ErrorPrint(err)
			} else {
				err = obj.Evaluator.Rotate(mainCipher, -obj.convMap[obj.lastFilterTreeDepth][shift+2], tempCtLv1)
				// obj.rot_num++
				ErrorPrint(err)
				err = obj.Evaluator.Add(mainCipher, tempCtLv1, mainCipher)
				ErrorPrint(err)
			}
			shift++
		}

		//opType2, split
		for s := 0; s < obj.splitNum; s++ {
			if i == 0 {
				tempCt, err := obj.Evaluator.MulNew(mainCipher, obj.lastFilter[i][s])
				ErrorPrint(err)
				splitedCiphertext = append(splitedCiphertext, tempCt)

			} else {
				tempCt, err := obj.Evaluator.MulNew(mainCipher, obj.lastFilter[i][s])
				ErrorPrint(err)

				err = obj.Evaluator.Add(splitedCiphertext[s], tempCt, splitedCiphertext[s])
				ErrorPrint(err)

			}
		}
	}
	//Rescaling
	for s := 0; s < obj.splitNum; s++ {
		err = obj.Evaluator.Rescale(splitedCiphertext[s], splitedCiphertext[s])
		ErrorPrint(err)
	}

	for i := 1; i < obj.convMap[obj.lastFilterTreeDepth+1][1]; i++ {
		err = obj.Evaluator.Rotate(splitedCiphertext[i], obj.convMap[obj.lastFilterTreeDepth+1][i+1], splitedCiphertext[i])
		// obj.rot_num++
		ErrorPrint(err)
		err = obj.Evaluator.Add(splitedCiphertext[0], splitedCiphertext[i], splitedCiphertext[0])
		ErrorPrint(err)
	}

	//copy paste
	for treeDepth := obj.lastFilterTreeDepth + 2; treeDepth < len(obj.convMap); treeDepth++ {
		if obj.convMap[treeDepth][0] == 0 {
			err = obj.Evaluator.Rotate(splitedCiphertext[0], obj.convMap[treeDepth][1], mainCipher)
			// obj.rot_num++
			ErrorPrint(err)
			err = obj.Evaluator.Add(splitedCiphertext[0], mainCipher, splitedCiphertext[0])
			ErrorPrint(err)
		} else {
			fmt.Println("Something wrong.. in conv.")
		}
	}

	// fmt.Println("rot num: ", obj.rot_num) //원
	return splitedCiphertext[0]
}

func RotOptConvRegister(convID string, depth int) [][]int {

	rotateSets := make([]map[int]bool, depth+1)

	for d := 0; d < depth+1; d++ {
		rotateSets[d] = make(map[int]bool)
	}

	convMap, _, rotIndex3by3Kernel := GetConvBlueprints(convID, depth)

	//combine rot index used in Conv map
	curDepth := 0
	for d := len(convMap) - 1; d >= 0; d-- {
		opType := convMap[d][0]
		if opType == 0 {
			rotateSets[curDepth][convMap[d][1]] = true
		} else if opType == 1 || opType == 2 {
			curDepth++
			for i := 2; i < len(convMap[d]); i++ {
				rotateSets[curDepth][convMap[d][i]] = true
				rotateSets[curDepth][-convMap[d][i]] = true
			}

		} else if opType == 3 {
			for i := 2; i < len(convMap[d]); i++ {
				rotateSets[curDepth][convMap[d][i]] = true
			}
		}
	}

	//combine rot index used in rotIndex3by3Kernel
	for i := 0; i < len(rotIndex3by3Kernel); i++ {
		rotateSets[depth][rotIndex3by3Kernel[i]] = true
	}

	rotateArray := make([][]int, depth+1)
	for d := 0; d < depth+1; d++ {
		rotateArray[d] = make([]int, 0)
		for element := range rotateSets[d] {
			if element != 0 {
				rotateArray[d] = append(rotateArray[d], element)
			}
		}
	}

	return rotateArray

}
