package core

import (
	"fmt"
	"strconv"

	"github.com/tuneinsight/lattigo/v5/core/rlwe"
	"github.com/tuneinsight/lattigo/v5/schemes/ckks"
)

// ///////////////////////////////////////////////////////////////////////////////////////////////////////
// Implementation of Rotation Optimized Convolution.
// ///////////////////////////////////////////////////////////////////////////////////////////////////////
type RotOptConv struct {
	encoder *ckks.Encoder

	Evaluator           *ckks.Evaluator
	params              ckks.Parameters
	preCompKernel       [][]*rlwe.Plaintext
	preCompBNadd        *rlwe.Plaintext
	preCompFilter       [][]*rlwe.Plaintext
	lastFilter          [][]*rlwe.Plaintext
	lastFilterTreeDepth int
	opType0TreeDepth    int
	dacToFor            []int
	dacToForTreeDepth   []int
	cf                  *ConvFeature

	layerNum           int
	blockNum           int
	operationNum       int
	convMap            [][]int
	q                  int //length of kernel_map
	rotIndex3by3Kernel []int

	splitNum int
	depth    int
}

func NewrotOptConv(ev *ckks.Evaluator, ec *ckks.Encoder, params ckks.Parameters, resnetLayerNum int, convID string, depth int, blockNum int, operationNum int) *RotOptConv {
	//rotOptConv Setting
	convMap, q, rotIndex3by3Kernel := GetConvBlueprints(convID, depth)

	// conv feature
	cf := GetConvFeature(convID)

	// plaintext setting, kernel weight
	path := "mulParModules/precomputed/rotOptConv/kernelWeight/" + strconv.Itoa(resnetLayerNum) + "/" + cf.LayerStr + "/" + strconv.Itoa(blockNum) + "/"
	var preCompKernel [][]*rlwe.Plaintext
	var preCompBNadd *rlwe.Plaintext
	var preCompFilter [][]*rlwe.Plaintext
	var lastFilter [][]*rlwe.Plaintext

	// preCompKernel generate
	filePath := path + "conv" + strconv.Itoa(operationNum) + "_weight"
	for i := 0; i < len(cf.KernelBP); i++ {
		var temp []*rlwe.Plaintext
		for j := 0; j < 9; j++ {
			temp = append(temp, txtToPlain(ec, filePath+strconv.Itoa(i)+"_"+strconv.Itoa(j)+".txt", params))
		}
		preCompKernel = append(preCompKernel, temp)
	}

	// preCompBNadd generate
	// filePath = path + "bn" + strconv.Itoa(operationNum) + "_add.txt"
	// preCompBNadd = txtToPlain(ec, filePath, params)

	// preCompFilter, lastFilter generate
	preCompFilter, lastFilter = MakeTxtRotOptConvFilter(convID, depth, ec, params)

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

	//get dacToFor values.
	var dacToFor []int
	var dacToForTreeDepth []int
	for depth := opType0TreeDepth; depth > 0; depth-- {
		opType := convMap[depth][0]
		if opType == 1 {
			dacToFor = append(dacToFor, convMap[depth][1])
			dacToForTreeDepth = append(dacToForTreeDepth, depth)
		}
	}
	for i := len(dacToFor); i < 3; i++ {
		dacToFor = append(dacToFor, 1)
	}
	for i := len(dacToForTreeDepth); i < 3; i++ {
		dacToForTreeDepth = append(dacToForTreeDepth, 0)
	}

	return &RotOptConv{
		encoder: ec,

		Evaluator:           ev,
		params:              params,
		preCompKernel:       preCompKernel,
		preCompBNadd:        preCompBNadd,
		preCompFilter:       preCompFilter,
		lastFilter:          lastFilter,
		lastFilterTreeDepth: lastFilterLocate,
		opType0TreeDepth:    opType0TreeDepth,
		dacToFor:            dacToFor,
		dacToForTreeDepth:   dacToForTreeDepth,
		cf:                  cf,
		splitNum:            splitNum,

		layerNum:           resnetLayerNum,
		blockNum:           blockNum,
		operationNum:       operationNum,
		convMap:            convMap,
		q:                  q,
		rotIndex3by3Kernel: rotIndex3by3Kernel,
		depth:              depth,
	}
}

func (obj RotOptConv) Foward2depth(ctIn *rlwe.Ciphertext) (ctOut *rlwe.Ciphertext) {
	mainCipher := ckks.NewCiphertext(obj.params, 1, ctIn.Level())
	tempCtLv1 := ckks.NewCiphertext(obj.params, 1, ctIn.Level())

	var err error
	// Rotate Data
	var rotInput []*rlwe.Ciphertext
	for w := 0; w < 9; w++ {
		c, err := obj.Evaluator.RotateNew(ctIn, obj.rotIndex3by3Kernel[w])
		ErrorPrint(err)
		rotInput = append(rotInput, c)
	}

	// conv
	var splitedCiphertext []*rlwe.Ciphertext
	for i := 0; i < obj.cf.q; i++ {
		kernelResult, err := obj.Evaluator.MulNew(rotInput[0], obj.preCompKernel[i][0])
		ErrorPrint(err)
		for w := 1; w < 9; w++ {
			tempCt, err := obj.Evaluator.MulNew(rotInput[w], obj.preCompKernel[i][w])
			ErrorPrint(err)
			err = obj.Evaluator.Add(kernelResult, tempCt, kernelResult)
			ErrorPrint(err)
		}
		err = obj.Evaluator.Rescale(kernelResult, mainCipher)
		ErrorPrint(err)

		//opType 0
		for treeDepth := obj.opType0TreeDepth; treeDepth < obj.lastFilterTreeDepth; treeDepth++ {
			err = obj.Evaluator.Rotate(mainCipher, obj.convMap[treeDepth][1], tempCtLv1)
			ErrorPrint(err)
			err = obj.Evaluator.Add(mainCipher, tempCtLv1, mainCipher)
			ErrorPrint(err)
		}

		// opType2, rotate
		shift := 0
		for j := 1; j < obj.convMap[obj.lastFilterTreeDepth][1]; j *= 2 {
			if ((i >> shift) & 1) == 0 {
				err = obj.Evaluator.Rotate(mainCipher, obj.convMap[obj.lastFilterTreeDepth][shift+2], tempCtLv1)
				ErrorPrint(err)
				err = obj.Evaluator.Add(mainCipher, tempCtLv1, mainCipher)
				ErrorPrint(err)
			} else {
				err = obj.Evaluator.Rotate(mainCipher, -obj.convMap[obj.lastFilterTreeDepth][shift+2], tempCtLv1)
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
		ErrorPrint(err)
		err = obj.Evaluator.Add(splitedCiphertext[0], splitedCiphertext[i], splitedCiphertext[0])
		ErrorPrint(err)
	}

	//copy paste
	for treeDepth := obj.lastFilterTreeDepth + 2; treeDepth < len(obj.convMap); treeDepth++ {
		if obj.convMap[treeDepth][0] == 0 {
			err = obj.Evaluator.Rotate(splitedCiphertext[0], obj.convMap[treeDepth][1], mainCipher)
			ErrorPrint(err)
			err = obj.Evaluator.Add(splitedCiphertext[0], mainCipher, splitedCiphertext[0])
			ErrorPrint(err)
		} else {
			fmt.Println("Something wrong.. in conv.")
		}
	}

	//Add bn_add
	// ctOut, err = obj.Evaluator.AddNew(splitedCiphertext[0], obj.preCompBNadd)
	// ErrorPrint(err)

	return ctOut
}
func (obj RotOptConv) Foward(ctIn *rlwe.Ciphertext) (ctOut *rlwe.Ciphertext) {
	if obj.depth == 2 {
		return obj.Foward2depth(ctIn) //2 depth consuming rotation optimized convolution.
	}

	mainCipher := ckks.NewCiphertext(obj.params, 1, ctIn.Level())
	mainCipherTemp := ckks.NewCiphertext(obj.params, 1, ctIn.Level())
	tempCtLv1 := ckks.NewCiphertext(obj.params, 1, ctIn.Level())
	tempD1 := ckks.NewCiphertext(obj.params, 1, ctIn.Level())
	tempD2 := ckks.NewCiphertext(obj.params, 1, ctIn.Level())
	tempD3 := ckks.NewCiphertext(obj.params, 1, ctIn.Level())

	d2Result := ckks.NewCiphertext(obj.params, 1, ctIn.Level())
	d2ResultTemp := ckks.NewCiphertext(obj.params, 1, ctIn.Level())
	d3Result := ckks.NewCiphertext(obj.params, 1, ctIn.Level())
	d3ResultTemp := ckks.NewCiphertext(obj.params, 1, ctIn.Level())

	var err error
	// Rotate Data
	var rotInput []*rlwe.Ciphertext
	for w := 0; w < 9; w++ {
		c, err := obj.Evaluator.RotateNew(ctIn, obj.rotIndex3by3Kernel[w])
		ErrorPrint(err)
		rotInput = append(rotInput, c)
	}

	// conv
	kernelNum := len(obj.cf.KernelBP) //all size
	beforeLastFilter := kernelNum / obj.convMap[obj.lastFilterTreeDepth][1]
	var splitedCiphertext []*rlwe.Ciphertext
	for i := 0; i < obj.convMap[obj.lastFilterTreeDepth][1]; i++ {

		// Naive implementation to avoid unnecessary copy for using divide and conquer
		startKernel := beforeLastFilter * i
		for d1 := 0; d1 < obj.dacToFor[0]; d1++ {
			for d2 := 0; d2 < obj.dacToFor[1]; d2++ {
				for d3 := 0; d3 < obj.dacToFor[2]; d3++ {
					//SISO convolution
					curKernel := startKernel + d1*obj.dacToFor[1]*obj.dacToFor[2] + d2*obj.dacToFor[2] + d3

					kernelResult, err := obj.Evaluator.MulNew(rotInput[0], obj.preCompKernel[curKernel][0])
					ErrorPrint(err)

					for w := 1; w < 9; w++ {
						tempCt, err := obj.Evaluator.MulNew(rotInput[w], obj.preCompKernel[curKernel][w])
						ErrorPrint(err)

						err = obj.Evaluator.Add(kernelResult, tempCt, kernelResult)
						ErrorPrint(err)
					}

					err = obj.Evaluator.Rescale(kernelResult, kernelResult)
					ErrorPrint(err)

					//Rot and combine
					dBack := 2
					if obj.dacToFor[dBack] != 1 {
						//rotate and add
						shift := 0
						for j := 1; j < obj.convMap[obj.dacToForTreeDepth[dBack]][1]; j *= 2 {
							if ((d3 >> shift) & 1) == 0 {
								err = obj.Evaluator.Rotate(kernelResult, obj.convMap[obj.dacToForTreeDepth[dBack]][shift+2], tempD3)
								ErrorPrint(err)
								err = obj.Evaluator.Add(kernelResult, tempD3, kernelResult)
								ErrorPrint(err)
							} else {
								err = obj.Evaluator.Rotate(kernelResult, -obj.convMap[obj.dacToForTreeDepth[dBack]][shift+2], tempD3)
								ErrorPrint(err)
								err = obj.Evaluator.Add(kernelResult, tempD3, kernelResult)
								ErrorPrint(err)
							}
							shift++
						}
						//filter and combine
						if d3 == 0 {
							err := obj.Evaluator.Mul(kernelResult, obj.preCompFilter[obj.dacToForTreeDepth[dBack]][d3], d3ResultTemp)
							ErrorPrint(err)

						} else {
							tempCt, err := obj.Evaluator.MulNew(kernelResult, obj.preCompFilter[obj.dacToForTreeDepth[dBack]][d3])
							ErrorPrint(err)

							err = obj.Evaluator.Add(tempCt, d3ResultTemp, d3ResultTemp)
							ErrorPrint(err)
						}
						err := obj.Evaluator.Rescale(d3ResultTemp, d3Result)
						ErrorPrint(err)

					} else {
						*d3Result = *kernelResult
					}
				}

				dBack := 1
				if obj.dacToFor[dBack] != 1 {
					// rotate and add
					shift := 0
					for j := 1; j < obj.convMap[obj.dacToForTreeDepth[dBack]][1]; j *= 2 {
						if ((d2 >> shift) & 1) == 0 {
							err = obj.Evaluator.Rotate(d3Result, obj.convMap[obj.dacToForTreeDepth[dBack]][shift+2], tempD2)
							ErrorPrint(err)
							err = obj.Evaluator.Add(d3Result, tempD2, d3Result)
							ErrorPrint(err)
						} else {
							err = obj.Evaluator.Rotate(d3Result, -obj.convMap[obj.dacToForTreeDepth[dBack]][shift+2], tempD2)
							ErrorPrint(err)
							err = obj.Evaluator.Add(d3Result, tempD2, d3Result)
							ErrorPrint(err)
						}
						shift++
					}
					// filter and combine
					if d2 == 0 {
						err := obj.Evaluator.Mul(d3Result, obj.preCompFilter[obj.dacToForTreeDepth[dBack]][d2], d2ResultTemp)
						ErrorPrint(err)

					} else {
						tempCt, err := obj.Evaluator.MulNew(d3Result, obj.preCompFilter[obj.dacToForTreeDepth[dBack]][d2])
						ErrorPrint(err)

						err = obj.Evaluator.Add(tempCt, d2ResultTemp, d2ResultTemp)
						ErrorPrint(err)
					}

					err = obj.Evaluator.Rescale(d2ResultTemp, d2Result)
					ErrorPrint(err)
				} else {
					*d2Result = *d3Result
				}
			}

			dBack := 0
			if obj.dacToFor[dBack] != 1 {
				// rotate and add
				shift := 0
				for j := 1; j < obj.convMap[obj.dacToForTreeDepth[dBack]][1]; j *= 2 {
					if ((d1 >> shift) & 1) == 0 {
						err = obj.Evaluator.Rotate(d2Result, obj.convMap[obj.dacToForTreeDepth[dBack]][shift+2], tempD1)
						ErrorPrint(err)
						err = obj.Evaluator.Add(d2Result, tempD1, d2Result)
						ErrorPrint(err)
					} else {
						err = obj.Evaluator.Rotate(d2Result, -obj.convMap[obj.dacToForTreeDepth[dBack]][shift+2], tempD1)
						ErrorPrint(err)
						err = obj.Evaluator.Add(d2Result, tempD1, d2Result)
						ErrorPrint(err)
					}
					shift++
				}
				// filter and combine
				if d1 == 0 {
					err := obj.Evaluator.Mul(d2Result, obj.preCompFilter[obj.dacToForTreeDepth[dBack]][d1], mainCipherTemp)
					ErrorPrint(err)
				} else {
					tempCt, err := obj.Evaluator.MulNew(d2Result, obj.preCompFilter[obj.dacToForTreeDepth[dBack]][d1])
					ErrorPrint(err)

					err = obj.Evaluator.Add(tempCt, mainCipherTemp, mainCipherTemp)
					ErrorPrint(err)
				}
				err = obj.Evaluator.Rescale(mainCipherTemp, mainCipher)
				ErrorPrint(err)

			} else {
				*mainCipher = *d2Result
			}
		}

		//opType 0
		for treeDepth := obj.opType0TreeDepth; treeDepth < obj.lastFilterTreeDepth; treeDepth++ {
			err = obj.Evaluator.Rotate(mainCipher, obj.convMap[treeDepth][1], tempCtLv1)
			ErrorPrint(err)
			err = obj.Evaluator.Add(mainCipher, tempCtLv1, mainCipher)
			ErrorPrint(err)
		}

		// opType2, rotate
		shift := 0
		for j := 1; j < obj.convMap[obj.lastFilterTreeDepth][1]; j *= 2 {
			if ((i >> shift) & 1) == 0 {
				err = obj.Evaluator.Rotate(mainCipher, obj.convMap[obj.lastFilterTreeDepth][shift+2], tempCtLv1)
				ErrorPrint(err)
				err = obj.Evaluator.Add(mainCipher, tempCtLv1, mainCipher)
				ErrorPrint(err)
			} else {
				err = obj.Evaluator.Rotate(mainCipher, -obj.convMap[obj.lastFilterTreeDepth][shift+2], tempCtLv1)
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
		ErrorPrint(err)
		err = obj.Evaluator.Add(splitedCiphertext[0], splitedCiphertext[i], splitedCiphertext[0])
		ErrorPrint(err)
	}

	//copy paste
	for treeDepth := obj.lastFilterTreeDepth + 2; treeDepth < len(obj.convMap); treeDepth++ {
		if obj.convMap[treeDepth][0] == 0 {
			err = obj.Evaluator.Rotate(splitedCiphertext[0], obj.convMap[treeDepth][1], mainCipher)
			ErrorPrint(err)
			err = obj.Evaluator.Add(splitedCiphertext[0], mainCipher, splitedCiphertext[0])
			ErrorPrint(err)
		} else {
			fmt.Println("Something wrong.. in conv.")
		}
	}

	//Add bn_add
	// ctOut, err = obj.Evaluator.AddNew(splitedCiphertext[0], obj.preCompBNadd)
	// ErrorPrint(err)

	return ctOut
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
