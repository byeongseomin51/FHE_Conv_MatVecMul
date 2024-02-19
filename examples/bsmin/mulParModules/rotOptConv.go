package mulParModules

import (
	"fmt"
	"strconv"
	"time"

	"github.com/tuneinsight/lattigo/v5/core/rlwe"
	"github.com/tuneinsight/lattigo/v5/schemes/ckks"
)

type RotOptConv struct {
	//for debugging
	encoder   *ckks.Encoder
	decryptor *rlwe.Decryptor

	Evaluator           *ckks.Evaluator
	params              ckks.Parameters
	preCompKernel       [][]*rlwe.Plaintext
	preCompBNadd        *rlwe.Plaintext
	preCompFilter       [][]*rlwe.Plaintext
	lastFilter          [][]*rlwe.Plaintext
	lastFilterTreeDepth int
	mode0TreeDepth      int
	dacToFor            []int
	dacToForTreeDepth   []int
	cf                  *ConvFeature

	layerNum           int
	blockNum           int
	operationNum       int
	convMap            [][]int
	q                  int //length of kernel_map
	rotIndex3by3Kernel []int
	beforeSplitNum     int
	splitNum           int
}

func NewrotOptConv(ev *ckks.Evaluator, ec *ckks.Encoder, dc *rlwe.Decryptor, params ckks.Parameters, resnetLayerNum int, convID string, depth int, blockNum int, operationNum int) *RotOptConv {
	// fmt.Println("Conv : ", resnetLayerNum, convID, depth, blockNum, operationNum)

	//rotOptConv Setting
	convMap, q, rotIndex3by3Kernel := GetConvMap(convID, depth)

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
	for i := 0; i < len(cf.KernelMap); i++ {
		var temp []*rlwe.Plaintext
		for j := 0; j < 9; j++ {
			temp = append(temp, txtToPlain(ec, filePath+strconv.Itoa(i)+"_"+strconv.Itoa(j)+".txt", params))
		}
		preCompKernel = append(preCompKernel, temp)
	}

	// preCompBNadd generate
	filePath = path + "bn" + strconv.Itoa(operationNum) + "_add.txt"
	preCompBNadd = txtToPlain(ec, filePath, params)

	// preCompFilter, lastFilter generate
	preCompFilter, lastFilter = MakeTxtRotOptConvFilter(convID, depth, ec, params)

	// get splitNum, lastFilterLocate, mode0TreeDepth value
	splitNum := 0
	lastFilterLocate := 0
	mode0TreeDepth := 0 //last mode0 or mode2 treeDepth locate
	for depth := len(convMap) - 1; depth > 0; depth-- {
		mode := convMap[depth][0]
		if mode == 2 {
			lastFilterLocate = depth
			mode0TreeDepth = depth
		} else if mode == 3 {
			splitNum = convMap[depth][1]
		} else if mode == 0 {
			mode0TreeDepth = depth
		} else {

		}
	}

	//get dacToFor values.
	var dacToFor []int
	var dacToForTreeDepth []int
	for depth := mode0TreeDepth; depth > 0; depth-- {
		mode := convMap[depth][0]
		if mode == 1 {
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
		encoder:   ec,
		decryptor: dc,

		Evaluator:           ev,
		params:              params,
		preCompKernel:       preCompKernel,
		preCompBNadd:        preCompBNadd,
		preCompFilter:       preCompFilter,
		lastFilter:          lastFilter,
		lastFilterTreeDepth: lastFilterLocate,
		mode0TreeDepth:      mode0TreeDepth,
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
	}
}

//for debugging

func (this RotOptConv) printCipher(fileName string, ctIn *rlwe.Ciphertext) {

	plainIn := this.decryptor.DecryptNew(ctIn)
	floatIn := make([]float64, this.params.MaxSlots())
	this.encoder.Decode(plainIn, floatIn)

	floatToTxt(fileName+".txt", floatIn)

}

func (this RotOptConv) Foward(ctIn *rlwe.Ciphertext) (ctOut *rlwe.Ciphertext) {

	start := time.Now()
	mainCipher := ckks.NewCiphertext(this.params, 1, ctIn.Level())
	tempCtLv1 := ckks.NewCiphertext(this.params, 1, ctIn.Level())
	tempD1 := ckks.NewCiphertext(this.params, 1, ctIn.Level())
	tempD2 := ckks.NewCiphertext(this.params, 1, ctIn.Level())
	tempD3 := ckks.NewCiphertext(this.params, 1, ctIn.Level())
	kernelResult := ckks.NewCiphertext(this.params, 1, ctIn.Level())

	d2Result := ckks.NewCiphertext(this.params, 1, ctIn.Level())
	d3Result := ckks.NewCiphertext(this.params, 1, ctIn.Level())
	fmt.Println(time.Now().Sub(start))
	// tempCtLv0 := ckks.NewCiphertext(this.params, 1, ctIn.Level())

	var err error
	// Rotate Data
	var rotInput []*rlwe.Ciphertext
	for w := 0; w < 9; w++ {
		c, err := this.Evaluator.RotateNew(ctIn, this.rotIndex3by3Kernel[w])
		ErrorPrint(err)
		rotInput = append(rotInput, c)
	}

	// conv
	kernelNum := len(this.cf.KernelMap) //all size
	beforeLastFilter := kernelNum / this.convMap[this.lastFilterTreeDepth][1]
	var splitedCiphertext []*rlwe.Ciphertext
	for i := 0; i < this.convMap[this.lastFilterTreeDepth][1]; i++ {
		start = time.Now()
		//use dac Sum
		// mainCipher = this.dacSum(this.mode0TreeDepth-1, beforeLastFilter*i, beforeLastFilter*(i+1), rotInput)

		startKernel := beforeLastFilter * i
		for d1 := 0; d1 < this.dacToFor[0]; d1++ {
			for d2 := 0; d2 < this.dacToFor[1]; d2++ {
				for d3 := 0; d3 < this.dacToFor[2]; d3++ {

					//SISO convolution
					start := startKernel + d1*this.dacToFor[1]*this.dacToFor[2] + d2*this.dacToFor[2] + d3

					tempCt, err := this.Evaluator.MulRelinNew(rotInput[0], this.preCompKernel[start][0])
					ErrorPrint(err)
					err = this.Evaluator.Rescale(tempCt, kernelResult)
					ErrorPrint(err)

					for w := 1; w < 9; w++ {
						tempCt, err := this.Evaluator.MulRelinNew(rotInput[w], this.preCompKernel[start][w])
						ErrorPrint(err)
						err = this.Evaluator.Rescale(tempCt, tempCt)
						ErrorPrint(err)
						err = this.Evaluator.Add(kernelResult, tempCt, kernelResult)
						ErrorPrint(err)
					}

					//Rot and combine
					dBack := 2
					if this.dacToFor[dBack] != 1 {
						//rotate and add
						shift := 0
						for j := 1; j < this.convMap[this.dacToForTreeDepth[dBack]][1]; j *= 2 {
							if ((d3 >> shift) & 1) == 0 {
								err = this.Evaluator.Rotate(kernelResult, this.convMap[this.dacToForTreeDepth[dBack]][shift+2], tempD3)
								ErrorPrint(err)
								err = this.Evaluator.Add(kernelResult, tempD3, kernelResult)
								ErrorPrint(err)
							} else {
								err = this.Evaluator.Rotate(kernelResult, -this.convMap[this.dacToForTreeDepth[dBack]][shift+2], tempD3)
								ErrorPrint(err)
								err = this.Evaluator.Add(kernelResult, tempD3, kernelResult)
								ErrorPrint(err)
							}
							shift++
						}
						//filter and combine
						if d3 == 0 {
							tempCt, err := this.Evaluator.MulRelinNew(kernelResult, this.preCompFilter[this.dacToForTreeDepth[dBack]][d3])
							ErrorPrint(err)
							err = this.Evaluator.Rescale(tempCt, d3Result)
							ErrorPrint(err)
						} else {
							tempCt, err := this.Evaluator.MulRelinNew(kernelResult, this.preCompFilter[this.dacToForTreeDepth[dBack]][d3])
							ErrorPrint(err)
							err = this.Evaluator.Rescale(tempCt, tempCt)
							ErrorPrint(err)
							err = this.Evaluator.Add(tempCt, d3Result, d3Result)
							ErrorPrint(err)
						}

					} else {
						*d3Result = *kernelResult
					}
				}

				//결과 32만큼, -32 만큼 rot 하고 더하기!
				dBack := 1
				if this.dacToFor[dBack] != 1 {
					// rotate and add
					shift := 0
					for j := 1; j < this.convMap[this.dacToForTreeDepth[dBack]][1]; j *= 2 {
						if ((d2 >> shift) & 1) == 0 {
							err = this.Evaluator.Rotate(d3Result, this.convMap[this.dacToForTreeDepth[dBack]][shift+2], tempD2)
							ErrorPrint(err)
							err = this.Evaluator.Add(d3Result, tempD2, d3Result)
							ErrorPrint(err)
						} else {
							err = this.Evaluator.Rotate(d3Result, -this.convMap[this.dacToForTreeDepth[dBack]][shift+2], tempD2)
							ErrorPrint(err)
							err = this.Evaluator.Add(d3Result, tempD2, d3Result)
							ErrorPrint(err)
						}
						shift++
					}
					// filter and combine
					if d2 == 0 {
						tempCt, err := this.Evaluator.MulRelinNew(d3Result, this.preCompFilter[this.dacToForTreeDepth[dBack]][d2])
						ErrorPrint(err)
						err = this.Evaluator.Rescale(tempCt, d2Result)
						ErrorPrint(err)
					} else {
						tempCt, err := this.Evaluator.MulRelinNew(d3Result, this.preCompFilter[this.dacToForTreeDepth[dBack]][d2])
						ErrorPrint(err)
						err = this.Evaluator.Rescale(tempCt, tempCt)
						ErrorPrint(err)
						err = this.Evaluator.Add(tempCt, d2Result, d2Result)
						ErrorPrint(err)
					}
				} else {
					*d2Result = *d3Result
				}
			}

			//1024, -1024 더하기!
			dBack := 0
			if this.dacToFor[dBack] != 1 {
				// rotate and add
				shift := 0
				for j := 1; j < this.convMap[this.dacToForTreeDepth[dBack]][1]; j *= 2 {
					if ((d1 >> shift) & 1) == 0 {
						err = this.Evaluator.Rotate(d2Result, this.convMap[this.dacToForTreeDepth[dBack]][shift+2], tempD1)
						ErrorPrint(err)
						err = this.Evaluator.Add(d2Result, tempD1, d2Result)
						ErrorPrint(err)
					} else {
						err = this.Evaluator.Rotate(d2Result, -this.convMap[this.dacToForTreeDepth[dBack]][shift+2], tempD1)
						ErrorPrint(err)
						err = this.Evaluator.Add(d2Result, tempD1, d2Result)
						ErrorPrint(err)
					}
					shift++
				}
				// filter and combine
				if d1 == 0 {
					tempCt, err := this.Evaluator.MulRelinNew(d2Result, this.preCompFilter[this.dacToForTreeDepth[dBack]][d1])
					ErrorPrint(err)
					err = this.Evaluator.Rescale(tempCt, mainCipher)
					ErrorPrint(err)
				} else {
					tempCt, err := this.Evaluator.MulRelinNew(d2Result, this.preCompFilter[this.dacToForTreeDepth[dBack]][d1])
					ErrorPrint(err)
					err = this.Evaluator.Rescale(tempCt, tempCt)
					ErrorPrint(err)
					err = this.Evaluator.Add(tempCt, mainCipher, mainCipher)
					ErrorPrint(err)
				}
			} else {
				*mainCipher = *d2Result
			}
		}

		fmt.Println(time.Now().Sub(start))
		start = time.Now()

		//mode 0
		for treeDepth := this.mode0TreeDepth; treeDepth < this.lastFilterTreeDepth; treeDepth++ {
			err = this.Evaluator.Rotate(mainCipher, this.convMap[treeDepth][1], tempCtLv1)
			ErrorPrint(err)
			err = this.Evaluator.Add(mainCipher, tempCtLv1, mainCipher)
			ErrorPrint(err)
		}

		fmt.Println(time.Now().Sub(start))
		start = time.Now()

		// mode2, rotate
		shift := 0
		for j := 1; j < this.convMap[this.lastFilterTreeDepth][1]; j *= 2 {
			if ((i >> shift) & 1) == 0 {
				err = this.Evaluator.Rotate(mainCipher, this.convMap[this.lastFilterTreeDepth][shift+2], tempCtLv1)
				ErrorPrint(err)
				err = this.Evaluator.Add(mainCipher, tempCtLv1, mainCipher)
				ErrorPrint(err)
			} else {
				err = this.Evaluator.Rotate(mainCipher, -this.convMap[this.lastFilterTreeDepth][shift+2], tempCtLv1)
				ErrorPrint(err)
				err = this.Evaluator.Add(mainCipher, tempCtLv1, mainCipher)
				ErrorPrint(err)
			}
			shift++
		}
		fmt.Println(time.Now().Sub(start))
		start = time.Now()
		//mode2, split
		for s := 0; s < this.splitNum; s++ {
			if i == 0 {
				tempCt, err := this.Evaluator.MulRelinNew(mainCipher, this.lastFilter[i][s])
				ErrorPrint(err)
				err = this.Evaluator.Rescale(tempCt, tempCt)
				ErrorPrint(err)
				splitedCiphertext = append(splitedCiphertext, tempCt)
			} else {
				tempCt, err := this.Evaluator.MulRelinNew(mainCipher, this.lastFilter[i][s])
				ErrorPrint(err)
				err = this.Evaluator.Rescale(tempCt, tempCt)
				ErrorPrint(err)
				err = this.Evaluator.Add(splitedCiphertext[s], tempCt, splitedCiphertext[s])
				ErrorPrint(err)
			}
		}
		fmt.Println(time.Now().Sub(start))
		start = time.Now()
	}

	// mode 3
	fmt.Println("mode3")
	start = time.Now()
	for i := 1; i < this.convMap[this.lastFilterTreeDepth+1][1]; i++ {
		err = this.Evaluator.Rotate(splitedCiphertext[i], this.convMap[this.lastFilterTreeDepth+1][i+1], splitedCiphertext[i])
		ErrorPrint(err)
		err = this.Evaluator.Add(splitedCiphertext[0], splitedCiphertext[i], splitedCiphertext[0])
		ErrorPrint(err)
	}
	fmt.Println(time.Now().Sub(start))
	start = time.Now()
	//copy paste
	for treeDepth := this.lastFilterTreeDepth + 2; treeDepth < len(this.convMap); treeDepth++ {
		if this.convMap[treeDepth][0] == 0 {
			err = this.Evaluator.Rotate(splitedCiphertext[0], this.convMap[treeDepth][1], mainCipher)
			ErrorPrint(err)
			err = this.Evaluator.Add(splitedCiphertext[0], mainCipher, splitedCiphertext[0])
			ErrorPrint(err)
		} else {
			fmt.Println("Something wrong.. in conv.")
		}
	}

	//Add bn_add
	ctOut, err = this.Evaluator.AddNew(splitedCiphertext[0], this.preCompBNadd)
	ErrorPrint(err)

	return ctOut
}

// func (this RotOptConv) dacSum(treeDepth, start, end int, rotInput []*rlwe.Ciphertext) (result *rlwe.Ciphertext) {
// 	mainCipher := ckks.NewCiphertext(this.params, rotInput[0].Degree(), rotInput[0].Level())
// 	tempCtLv1 := ckks.NewCiphertext(this.params, rotInput[0].Degree(), rotInput[0].Level())
// 	tempCtLv0 := ckks.NewCiphertext(this.params, rotInput[0].Degree(), rotInput[0].Level())
// 	result = ckks.NewCiphertext(this.params, rotInput[0].Degree(), rotInput[0].Level())

// 	var err error
// 	if treeDepth == 0 {
// 		tempCt, err := this.Evaluator.MulRelinNew(rotInput[0], this.preCompKernel[start][0])
// 		ErrorPrint(err)
// 		err = this.Evaluator.Rescale(tempCt, tempCt)
// 		result = tempCt
// 		ErrorPrint(err)

// 		for w := 1; w < 9; w++ {
// 			tempCt, err := this.Evaluator.MulRelinNew(rotInput[w], this.preCompKernel[start][w])
// 			ErrorPrint(err)
// 			err = this.Evaluator.Rescale(tempCt, tempCtLv1)
// 			ErrorPrint(err)
// 			err = this.Evaluator.Add(result, tempCtLv1, result)
// 			ErrorPrint(err)
// 		}
// 	} else {
// 		if this.convMap[treeDepth][0] != 1 { //not mode 0
// 			fmt.Println("Something wrong in dacSum..")
// 		} else {
// 			allLen := end - start
// 			minLen := allLen / this.convMap[treeDepth][1]

//				for i := 0; i < this.convMap[treeDepth][1]; i++ {
//					// get pre result
//					mainCipher = this.dacSum(treeDepth-1, minLen*i, minLen*(i+1), rotInput)
//					//rotate and add
//					shift := 0
//					for j := 1; j < this.convMap[treeDepth][1]; j *= 2 {
//						if ((i >> shift) & 1) == 0 {
//							err = this.Evaluator.Rotate(mainCipher, this.convMap[treeDepth][shift+2], tempCtLv1)
//							ErrorPrint(err)
//							err = this.Evaluator.Add(mainCipher, tempCtLv1, mainCipher)
//							ErrorPrint(err)
//						} else {
//							err = this.Evaluator.Rotate(mainCipher, -this.convMap[treeDepth][shift+2], tempCtLv1)
//							ErrorPrint(err)
//							err = this.Evaluator.Add(mainCipher, tempCtLv1, mainCipher)
//							ErrorPrint(err)
//						}
//						shift++
//					}
//					//filter and combine
//					if i == 0 {
//						tempCt, err := this.Evaluator.MulRelinNew(mainCipher, this.preCompFilter[treeDepth][i])
//						ErrorPrint(err)
//						err = this.Evaluator.Rescale(tempCt, result)
//						ErrorPrint(err)
//					} else {
//						tempCt, err := this.Evaluator.MulRelinNew(mainCipher, this.preCompFilter[treeDepth][i])
//						ErrorPrint(err)
//						err = this.Evaluator.Rescale(tempCt, tempCtLv0)
//						ErrorPrint(err)
//						err = this.Evaluator.Add(tempCtLv0, result, result)
//						ErrorPrint(err)
//					}
//				}
//			}
//		}
//		return result
//	}
func RotOptConvRegister(convID string, depth int) []int {

	rotateSets := make(map[int]bool)

	convMap, _, rotIndex3by3Kernel := GetConvMap(convID, depth)

	//combine rot index used in Conv map
	for d := 1; d < len(convMap); d++ {
		mode := convMap[d][0]
		if mode == 0 {
			rotateSets[convMap[d][1]] = true
		} else if mode == 1 || mode == 2 {
			for i := 2; i < len(convMap[d]); i++ {
				rotateSets[convMap[d][i]] = true
				rotateSets[-convMap[d][i]] = true
			}
		} else if mode == 3 {
			for i := 2; i < len(convMap[d]); i++ {
				rotateSets[convMap[d][i]] = true
			}
		}
	}

	//combine rot index used in rotIndex3by3Kernel
	for i := 0; i < len(rotIndex3by3Kernel); i++ {
		rotateSets[rotIndex3by3Kernel[i]] = true
	}
	var rotateArray []int
	for element := range rotateSets {
		if element != 0 {
			rotateArray = append(rotateArray, element)
		}
	}

	return rotateArray

}

func GetConvMap(convID string, depth int) ([][]int, int, []int) {
	var convMap [][]int
	var q int //length of kernel_map
	var rotIndex3by3Kernel []int

	if convID == "CONV1" { //32*32*3 -> 32*32*16, kernel=3*3, k=1
		//CONV1
		//=================Choose MAP=================//
		//2 depth, 14 rotation
		if depth == 2 {
			convMap = [][]int{ //1499ms
				{4}, //tree length
				{0, 2048},
				{2, 2, 1024},
				{3, 2, 14336},
				{0, -16384},
			}

		} else {
			fmt.Printf("RotOptConv : Invalid parameter! convID(%s), depth(%v)", convID, depth)
		}

		//============================================//
		q = 2
		rotIndex3by3Kernel = []int{-33, -32, -31, -1, 0, 1, 31, 32, 33}

		//========================================================================================//
	} else if convID == "CONV2" { //32*32*16 -> 32*32*16, kernel=3*3, k=1
		//CONV2
		//=================Choose MAP=================//
		if depth == 2 { //5089ms
			//2depth 36 rotations
			convMap = [][]int{
				{3}, //tree length
				{2, 8, 1024, 2048, 4096},
				{3, 4, 8192, 8192, 16384},
				{0, -16384},
			}
		} else if depth == 3 {
			//3 depth, 28 rotation
			convMap = [][]int{ //4224ms
				{4}, //tree length
				{1, 2, 1024},
				{2, 4, 2048, 4096},
				{3, 4, 8192, 8192, 16384},
				{0, -16384},
			}
		} else if depth == 4 {
			//4 depth, 26 rotation
			convMap = [][]int{ //3871ms
				{5}, //tree length
				{1, 2, 1024},
				{1, 2, 2048},
				{2, 2, 4096},
				{3, 4, 8192, 8192, 16384},
				{0, -16384},
			}
		} else {
			fmt.Printf("RotOptConv : Invalid parameter! convID(%s), depth(%v)", convID, depth)
		}

		//============================================//
		q = 8
		rotIndex3by3Kernel = []int{-33, -32, -31, -1, 0, 1, 31, 32, 33}

		//========================================================================================//
	} else if convID == "CONV3s2" { //32*32*16 -> 16*16*32, kernel=3*3, k=1->2
		//CONV3s2
		//=================Choose MAP=================//
		if depth == 2 {
			//2 depth 77 rotation
			convMap = [][]int{ //10854ms
				{4}, //tree length
				{2, 16, 1024, 2048, 4096, 8192},
				{3, 4, 8192 - 1, 16384 - 32, 16384 + 8192 - 32 - 1},
				{0, -8192},
				{0, -16384},
			}
		} else if depth == 3 {
			//3 depth, 53 rotation,
			convMap = [][]int{ //8212ms
				{5}, //tree length
				{1, 4, 1024, 2048},
				{2, 4, 4096, 8192},
				{3, 4, 8192 - 1, 16384 - 32, 16384 + 8192 - 32 - 1},
				{0, -8192},
				{0, -16384},
			}
		} else if depth == 4 {
			//4 depth, 45 rotation,
			convMap = [][]int{ //7688ms
				{6}, //tree length
				{1, 2, 1024},
				{1, 2, 2048},
				{2, 4, 4096, 8192},
				{3, 4, 8192 - 1, 16384 - 32, 16384 + 8192 - 32 - 1},
				{0, -8192},
				{0, -16384},
			}
		} else if depth == 5 {
			//5 depth, 43 rotation,
			convMap = [][]int{ //7688ms
				{7}, //tree length
				{1, 2, 1024},
				{1, 2, 2048},
				{1, 2, 4096},
				{2, 2, 8192},
				{3, 4, 8192 - 1, 16384 - 32, 16384 + 8192 - 32 - 1},
				{0, -8192},
				{0, -16384},
			}
		} else {
			fmt.Printf("RotOptConv : Invalid parameter! convID(%s), depth(%v)", convID, depth)

		}

		//============================================//
		q = 16
		rotIndex3by3Kernel = []int{-33, -32, -31, -1, 0, 1, 31, 32, 33}

		//========================================================================================//
	} else if convID == "CONV3" { //16*16*32 -> 16*16*32, kernel=3*3, k=2
		//CONV3
		//=================Choose MAP=================//
		if depth == 2 {
			//2 depth, 49 rotation
			convMap = [][]int{ //6630 ms
				{5}, //tree length
				{0, 2048},
				{2, 8, 1, 32, 1024},
				{3, 8, 4096, 4096*2 - 2048, 4096*3 - 2048, 4096*4 - 4096, 4096*5 - 4096, 4096*6 - 6144, 4096*7 - 6144}, // 이렇게 하면9167 ms, 10254 ms {3,16,2048,4096,6144,8192-2048,8192,8192+2048,8192+4096,16384-4096,16384-2048,16384,16384+2048,16384+2048,16384+4096,16384+6144,16384+8192},
				{0, -8192},
				{0, -16384},
			}
		} else if depth == 3 {
			//3 depth, 35 rotations
			convMap = [][]int{ //4583 ms
				{7}, //tree length
				{1, 4, 1, 32},
				{0, 2048},
				{0, 4096},
				{2, 2, 1024},
				{3, 4, 8192 - 2048, 16384 - 4096, 16384 + 2048},
				{0, -8192},
				{0, -16384},
			}
		} else if depth == 4 {
			//4 depth, 31 rotation
			convMap = [][]int{ //4247ms
				{8}, //tree length
				{1, 2, 1},
				{1, 2, 32},
				{0, 2048},
				{0, 4096},
				{2, 2, 1024},
				{3, 4, 8192 - 2048, 16384 - 4096, 16384 + 2048},
				{0, -8192},
				{0, -16384},
			}
		} else {
			fmt.Printf("RotOptConv : Invalid parameter! convID(%s), depth(%v)", convID, depth)

		}

		//============================================//
		q = 8
		rotIndex3by3Kernel = []int{-66, -64, -62, -2, 0, 2, 62, 64, 66}

		//========================================================================================//
	} else if convID == "CONV4s2" { //16*16*32 -> 8*8*64, kernel=3*3, k=2->4
		//CONV4s2
		//=================Choose MAP=================//
		if depth == 2 {
			//2 depth, 82 rotation
			convMap = [][]int{ //15231 ms
				{5}, //tree length
				{2, 16, 1, 32, 1024, 2048},
				{3, 8, 4096, 8192 - 2, 8192 - 2 + 4096, 16384 - 32 - 32, 16384 - 32 - 32 + 4096, 16384 + 8192 - 32 - 32 - 2, 16384 + 8192 - 32 - 32 - 2 + 4096},
				{0, -4096},
				{0, -8192},
				{0, -16384},
			}
		} else if depth == 3 {
			//3 depth, 58 rotation
			convMap = [][]int{ //8492ms
				{7},                //tree length
				{1, 4, 1, 32},      //+2*4 = 8
				{0, 4096},          //나중에 없앨 듯 9
				{2, 4, 1024, 2048}, //(9+2)*4=44
				{3, 4, 8192 - 2, 16384 - 32 - 32, 16384 + 8192 - 32 - 32 - 2}, //47
				{0, -4096},  //48
				{0, -8192},  //49
				{0, -16384}, //50
			}
		} else if depth == 4 {
			//4 depth, 50 rotation
			convMap = [][]int{ //7498ms [4,2,1]
				{8},              //tree length
				{1, 2, 1},        //2
				{1, 4, 32, 1024}, //16
				{0, 4096},        //17
				{2, 2, 2048},     //36
				{3, 4, 8192 - 2, 16384 - 32 - 32, 16384 + 8192 - 32 - 32 - 2}, //39
				{0, -4096},
				{0, -8192},
				{0, -16384}, //42
			}
		} else if depth == 5 {
			//5 depth, 46 rotation
			convMap = [][]int{ //7498ms
				{9}, //tree length
				{1, 2, 1},
				{1, 2, 32},
				{1, 2, 1024},
				{0, 4096},
				{2, 2, 2048},
				{3, 4, 8192 - 2, 16384 - 32 - 32, 16384 + 8192 - 32 - 32 - 2},
				{0, -4096},
				{0, -8192},
				{0, -16384},
			}
		} else {
			fmt.Printf("RotOptConv : Invalid parameter! convID(%s), depth(%v)", convID, depth)

		}

		//============================================//
		q = 16
		rotIndex3by3Kernel = []int{-66, -64, -62, -2, 0, 2, 62, 64, 66}

		//========================================================================================//
	} else if convID == "CONV4" { //8*8*64 -> 8*8*64, kernel=3*3, k=4
		//CONV4
		//=================Choose MAP=================//
		if depth == 2 {
			//OPTION 3 : 2 depth, 66 rotation
			convMap = [][]int{ //7906ms
				{8}, //tree length
				{0, 32 + 32},
				{0, 1024},
				{0, 2048},
				{2, 8, 1, 2, 32},
				{3, 8, 1024*4 - 64, 1024 * 7, 1024*11 - 64, 1024 * 14, 1024*18 - 64, 1024 * 21, 1024*25 - 64},
				{0, -4096},
				{0, -8192},
				{0, -16384},
			}
		} else if depth == 3 {
			//3 depth, 42 rotation
			convMap = [][]int{ //5299ms
				{9},          //tree length
				{1, 4, 1, 2}, //2*4=8
				{0, 32 + 32}, //9
				{0, 1024},    //10
				{0, 2048},    //11
				{2, 2, 32},   //(11+1)*2 = 24
				{3, 8, 1024*4 - 64, 1024 * 7, 1024*11 - 64, 1024 * 14, 1024*18 - 64, 1024 * 21, 1024*25 - 64},
				{0, -4096},  //32
				{0, -8192},  //33
				{0, -16384}, //34
			}
		} else if depth == 4 {
			//4 depth, 38 rotation
			convMap = [][]int{ //4809ms
				{10}, //tree length
				{1, 2, 1},
				{1, 2, 2},
				{0, 32 + 32},
				{0, 1024},
				{0, 2048},
				{2, 2, 32},
				{3, 8, 1024*4 - 64, 1024 * 7, 1024*11 - 64, 1024 * 14, 1024*18 - 64, 1024 * 21, 1024*25 - 64},
				{0, -4096},
				{0, -8192},
				{0, -16384},
			}
		} else {
			fmt.Printf("RotOptConv : Invalid parameter! convID(%s), depth(%v)", convID, depth)

		}
		//============================================//

		q = 8
		rotIndex3by3Kernel = []int{-132, -128, -124, -4, 0, 4, 124, 128, 132}
	}
	return convMap, q, rotIndex3by3Kernel
}
