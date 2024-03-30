package mulParModules

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tuneinsight/lattigo/v5/core/rlwe"
	"github.com/tuneinsight/lattigo/v5/schemes/ckks"
)

func reverse(slice []float64) {
	for i, j := 0, len(slice)-1; i < j; i, j = i+1, j-1 {
		slice[i], slice[j] = slice[j], slice[i]
	}
}
func cyclicShiftLeft(slice []float64, shift int) {
	length := len(slice)
	if length == 0 {
		return
	}
	shift %= length
	if shift < 0 {
		shift += length
	}
	reverse(slice[:shift])
	reverse(slice[shift:])
	reverse(slice)
}

func PlainRot(slice []float64, rotNum int) []float64 {
	if rotNum > 0 {
		cyclicShiftLeft(slice, rotNum)
	} else {
		cyclicShiftLeft(slice, len(slice)+rotNum)
	}
	return slice
}

func PlaintextRot(plaintext *rlwe.Plaintext, rotNum int, ec *ckks.Encoder, params ckks.Parameters) *rlwe.Plaintext {
	slice := make([]float64, 32768)
	ec.Decode(plaintext, slice)
	if rotNum > 0 {
		cyclicShiftLeft(slice, rotNum)
	} else {
		cyclicShiftLeft(slice, len(slice)+rotNum)
	}
	resultPlain := ckks.NewPlaintext(params, params.MaxLevel())
	ec.Encode(slice, resultPlain)
	return resultPlain
}

// Filter out
// parNum = which parallel data
// k = multiplexed number
// channel = which channel. channel begin from 0
func GeneralFilter(channel int, parNum int, k int) []float64 {
	result := make([]float64, 32768)
	allPar := 1
	allChannels := 1
	if k == 1 { // 32*32*16 = 2^14
		allPar = 2
		allChannels = 16
	} else if k == 2 { //32*32*8 = 2^13
		allPar = 4
		allChannels = 32
	} else if k == 4 { //32*32*4 =2^12
		allPar = 8
		allChannels = 64
	}

	if allChannels <= channel || allPar <= parNum || k > 4 {
		fmt.Println("Something wrong in GeneralFilter() !")
		return result
	}

	// 첫번째 채널에 해당하는 필터 만들음
	for h := 0; h < 32; h++ {
		for w := 0; w < 32; w++ {
			if w%k == 0 && h%k == 0 {
				result[w+h*32] = 1
				// fmt.Println(w + h*32)
			}
		}
	}

	// 그 채널을 channel 만큼 돌림.
	channelRot := -(1024*(channel/(k*k)) + 32*(channel%(k*k)/k) + (channel % (k * k) % k))
	result = PlainRot(result, channelRot)
	// fmt.Println("c : ", channelRot)

	// 그걸 또 parNum 만큼 돌림.
	parRot := -(32768 / allPar * parNum)
	result = PlainRot(result, parRot)
	// fmt.Println("p : ", parRot)
	return result
}

func MakeTxtRotOptConvFilter(convID string, depth int, encoder *ckks.Encoder, params ckks.Parameters) (preCompFilter [][]*rlwe.Plaintext, lastFilter [][]*rlwe.Plaintext) {

	//get ConvMap
	convMap, _, _ := GetConvMap(convID, depth)

	// get convFeature
	convFeatureMap := GetConvFeature(convID)

	//param setting
	preCompFilter = make([][]*rlwe.Plaintext, len(convMap))

	//one by one depth
	for treeDepth := 1; treeDepth < len(convMap); treeDepth++ {

		//get mode
		mode := convMap[treeDepth][0]

		//Make filter
		if mode == 1 {

			//Make filter
			filter := mode1Filter(convMap[treeDepth])

			//Save filter
			preCompFilter[treeDepth] = make([]*rlwe.Plaintext, len(filter))
			for num := 0; num < len(filter); num++ {
				preCompFilter[treeDepth][num] = floatToPlain(filter[num], encoder, params)
			}
		} else if mode == 2 {
			//declare
			var filterWithSplit [][][]float64

			//Make filter
			////first make mode 1 filter
			nonSplitFilter := mode1Filter(convMap[treeDepth])

			//// 앞에 있던 mode 0 고려
			for zeroCheck := treeDepth - 1; zeroCheck > 0; zeroCheck-- {
				if convMap[zeroCheck][0] == 0 {
					for t := 0; t < len(nonSplitFilter); t++ {
						nonSplitFilter[t] = multVec(nonSplitFilter[t], mode0Filter(convMap[zeroCheck][1]))
					}
				} else {
					break
				}
			}

			//// stride 고려
			if convFeatureMap.Stride != 1 {
				for t := 0; t < len(nonSplitFilter); t++ {
					nonSplitFilter[t] = multVec(nonSplitFilter[t], StrideFilter(convFeatureMap.K))
				}
			}

			//// split 고려
			splitNum := 0
			for tempD := len(convMap) - 1; tempD > 0; tempD-- {
				if convMap[tempD][0] == 3 {
					splitNum = convMap[tempD][1]
				}
			}
			for i := 0; i < len(nonSplitFilter); i++ {
				filterWithSplit = append(filterWithSplit, crossFilter(nonSplitFilter[i], splitNum))
			}

			//Save filter
			lastFilter = make([][]*rlwe.Plaintext, len(filterWithSplit))
			for num := 0; num < len(filterWithSplit); num++ {
				lastFilter[num] = make([]*rlwe.Plaintext, len(filterWithSplit[0]))
				for split := 0; split < len(filterWithSplit[0]); split++ {
					lastFilter[num][split] = floatToPlain(filterWithSplit[num][split], encoder, params)
				}
			}
		} else {
		} //mode0, 3 no filter

	}
	return preCompFilter, lastFilter
}
func StrideFilter(k int) []float64 {
	var strideFilter []float64

	if k == 1 {
		for ii := 0; ii < 32768/64; ii++ {
			for i := 0; i < 16; i++ {
				strideFilter = append(strideFilter, 1, 0)
			}
			for i := 0; i < 32; i++ {
				strideFilter = append(strideFilter, 0)
			}
		}
	} else if k == 2 {
		for ii := 0; ii < 32768/128; ii++ {
			for i := 0; i < 16; i++ {
				strideFilter = append(strideFilter, 1, 1, 0, 0)
			}
			for i := 0; i < 64; i++ {
				strideFilter = append(strideFilter, 0)
			}
		}
	}

	return strideFilter
}
func mode1Filter(treePart []int) [][]float64 {

	var resultFilter [][]float64
	for subnode := 0; subnode < treePart[1]; subnode++ {
		//declare
		shift := 0
		var eachFilter []float64
		for i := 0; i < 32768; i++ {
			eachFilter = append(eachFilter, 1)
		}
		//mult with mode0filter
		for i := 1; i < treePart[1]; i *= 2 {
			if ((subnode >> shift) & 1) == 0 {
				eachFilter = multVec(eachFilter, mode0Filter(treePart[shift+2]))
			} else {
				eachFilter = multVec(eachFilter, mode0Filter(-treePart[shift+2]))
			}
			shift++
		}
		//combine to resultFilter
		resultFilter = append(resultFilter, eachFilter)

	}

	return resultFilter

}

func mode0Filter(rotIndex int) []float64 {
	var resultFilter []float64

	if rotIndex > 0 {
		for i := 0; i < 32768; i++ {
			if (i/rotIndex)%2 == 0 { //+일 때 여기
				resultFilter = append(resultFilter, 1)
			} else { // - 일 때 여기
				resultFilter = append(resultFilter, 0)
			}
		}
	} else {
		for i := 0; i < 32768; i++ {
			if (i/rotIndex)%2 == 0 { //+일 때 여기
				resultFilter = append(resultFilter, 0)
			} else { // - 일 때 여기
				resultFilter = append(resultFilter, 1)
			}
		}
	}

	return resultFilter

}

func crossFilter(filter []float64, splitNum int) [][]float64 {
	var crossFilter [][]float64
	length := 32768
	for s := 0; s < splitNum; s++ {
		var splitTemp []float64
		validStart := (length / splitNum) * s
		validEnd := (length / splitNum) * (s + 1)
		for i := 0; i < length; i++ {
			if i >= validStart && i < validEnd {
				splitTemp = append(splitTemp, 1)
			} else {
				splitTemp = append(splitTemp, 0)
			}
		}

		crossFilter = append(crossFilter, multVec(splitTemp, filter))
	}

	return crossFilter
}

func LeftUpFilter(k int, isCONV1 bool) []float64 {
	var filter []float64
	b := 1
	if k == 1 && isCONV1 {
		b = 8
		for block := 0; block < b; block++ {
			for i := 0; i < 32768/b; i++ {
				if i < 1024 {
					filter = append(filter, 1)
				} else {
					filter = append(filter, 0)
				}
			}
		}
	} else if k == 1 && isCONV1 == false {
		b = 2
		for block := 0; block < b; block++ {
			for i := 0; i < 32768/b; i++ {
				if i < 1024 {
					filter = append(filter, 1)
				} else {
					filter = append(filter, 0)
				}
			}
		}
	} else if k == 2 {
		b = 4
		for block := 0; block < b; block++ {
			for i := 0; i < 32768/b; i++ {
				if i%k == 0 && (i/32)%k == 0 && i < 1024 {
					filter = append(filter, 1)
				} else {
					filter = append(filter, 0)
				}
			}
		}
	} else if k == 4 {
		b = 8
		for block := 0; block < b; block++ {
			for i := 0; i < 32768/b; i++ {
				if i%k == 0 && (i/32)%k == 0 && i < 1024 {
					filter = append(filter, 1)
				} else {
					filter = append(filter, 0)
				}
			}
		}
	}

	return filter
}

func MakeTxtRotOptConvWeight() {
	layerNums := []int{20, 32, 44, 56, 110}
	for _, layerNum := range layerNums {
		originalFolderPath := "mulParModules/precomputed/resnetPtParam/" + strconv.Itoa(layerNum) + "/"
		modifiedFolderPath := "mulParModules/precomputed/rotOptConv/kernelWeight/" + strconv.Itoa(layerNum) + "/"

		// Make kernel weight
		err := filepath.Walk(originalFolderPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			originalFileName := filepath.Base(path)
			nameSplited := strings.Split(originalFileName, "_")

			modifiedFilePath := modifiedFolderPath

			switch nameSplited[0] {
			case "layer1":
				//Make modified file path
				modifiedFilePath = modifiedFilePath + "layer1/" + nameSplited[1] + "/"
				for i := 2; i < len(nameSplited); i++ {
					modifiedFilePath += nameSplited[i]
					if i != len(nameSplited)-1 {
						modifiedFilePath += "_"
					}
				}
				//For Conv Param
				if nameSplited[2] == "conv1" || nameSplited[2] == "conv2" {
					var x int
					if nameSplited[2] == "conv1" {
						x = 1
					} else {
						x = 2
					}
					makeModifyKernel(originalFolderPath+originalFileName, modifiedFilePath, "CONV2", originalFolderPath+nameSplited[0]+"_"+nameSplited[1]+"_bn"+strconv.Itoa(x))
					//For BN param
				} else if (nameSplited[2] == "bn1" && nameSplited[3] == "bias.txt") || (nameSplited[2] == "bn2" && nameSplited[3] == "bias.txt") {
					makeBn(32, 1, originalFolderPath+nameSplited[0]+"_"+nameSplited[1]+"_"+nameSplited[2], modifiedFolderPath+nameSplited[0]+"/"+nameSplited[1]+"/"+nameSplited[2])
				}
			case "layer2":
				//Make modified file path
				modifiedFilePath = modifiedFilePath + "layer2/" + nameSplited[1] + "/"
				for i := 2; i < len(nameSplited); i++ {
					modifiedFilePath += nameSplited[i]
					if i != len(nameSplited)-1 {
						modifiedFilePath += "_"
					}
				}
				//For Conv Param
				if nameSplited[2] == "conv1" || nameSplited[2] == "conv2" {
					var x int
					if nameSplited[2] == "conv1" {
						x = 1
					} else {
						x = 2
					}
					if nameSplited[1] == "0" && nameSplited[2] == "conv1" {
						makeModifyKernel(originalFolderPath+originalFileName, modifiedFilePath, "CONV3s2", originalFolderPath+nameSplited[0]+"_"+nameSplited[1]+"_bn"+strconv.Itoa(x))
					} else {
						makeModifyKernel(originalFolderPath+originalFileName, modifiedFilePath, "CONV3", originalFolderPath+nameSplited[0]+"_"+nameSplited[1]+"_bn"+strconv.Itoa(x))
					}
					//For BN param
				} else if (nameSplited[2] == "bn1" && nameSplited[3] == "bias.txt") || (nameSplited[2] == "bn2" && nameSplited[3] == "bias.txt") {
					makeBn(16, 2, originalFolderPath+nameSplited[0]+"_"+nameSplited[1]+"_"+nameSplited[2], modifiedFolderPath+nameSplited[0]+"/"+nameSplited[1]+"/"+nameSplited[2])
				}

			case "layer3":
				//Make modified file path
				modifiedFilePath = modifiedFilePath + "layer3/" + nameSplited[1] + "/"
				for i := 2; i < len(nameSplited); i++ {
					modifiedFilePath += nameSplited[i]
					if i != len(nameSplited)-1 {
						modifiedFilePath += "_"
					}
				}
				//For Conv Param
				if nameSplited[2] == "conv1" || nameSplited[2] == "conv2" {
					var x int
					if nameSplited[2] == "conv1" {
						x = 1
					} else {
						x = 2
					}
					if nameSplited[1] == "0" && nameSplited[2] == "conv1" {
						makeModifyKernel(originalFolderPath+originalFileName, modifiedFilePath, "CONV4s2", originalFolderPath+nameSplited[0]+"_"+nameSplited[1]+"_bn"+strconv.Itoa(x))
					} else {
						makeModifyKernel(originalFolderPath+originalFileName, modifiedFilePath, "CONV4", originalFolderPath+nameSplited[0]+"_"+nameSplited[1]+"_bn"+strconv.Itoa(x))
					}
					//For BN param
				} else if (nameSplited[2] == "bn1" && nameSplited[3] == "bias.txt") || (nameSplited[2] == "bn2" && nameSplited[3] == "bias.txt") {
					makeBn(8, 4, originalFolderPath+nameSplited[0]+"_"+nameSplited[1]+"_"+nameSplited[2], modifiedFolderPath+nameSplited[0]+"/"+nameSplited[1]+"/"+nameSplited[2])
				}
			case "linear":
				// modifiedFilePath = modifiedFilePath + "linear/" + nameSplited[1]
				// if nameSplited[1] == "bias.txt" {
				// 	makeBias(originalFolderPath+originalFileName, modifiedFilePath)
				// } else if nameSplited[1] == "weight.txt" {
				// 	makeLinearWeight(originalFolderPath+originalFileName, modifiedFilePath)
				// }
			default:
				//Make modified file path
				modifiedFilePath += "layer0/0/"
				//For Conv Param
				if nameSplited[0] == "conv1" {
					modifiedFilePath += "conv1_weight.txt" //CONV1이라 특이하게 적용.
					makeModifyKernel(originalFolderPath+originalFileName, modifiedFilePath, "CONV1", originalFolderPath+"bn1")

					//For BN1 param
				} else if nameSplited[0] == "bn1" && nameSplited[1] == "bias.txt" {
					makeBn(32, 1, originalFolderPath+nameSplited[0], modifiedFilePath+"bn1")
				}
			}

			return nil
		})

		if err != nil {
			fmt.Println("오류:", err)
		}
	}

}

func MakeTxtMulParConvWeight() {
	layerNums := []int{20, 32, 44, 56, 110}
	for _, layerNum := range layerNums {
		originalFolderPath := "mulParModules/precomputed/resnetPtParam/" + strconv.Itoa(layerNum) + "/"
		modifiedFolderPath := "mulParModules/precomputed/mulParConv/kernelWeight/" + strconv.Itoa(layerNum) + "/"

		// Make kernel weight
		err := filepath.Walk(originalFolderPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			originalFileName := filepath.Base(path)
			nameSplited := strings.Split(originalFileName, "_")

			modifiedFilePath := modifiedFolderPath

			switch nameSplited[0] {
			case "layer1":
				//Make modified file path
				modifiedFilePath = modifiedFilePath + "layer1/" + nameSplited[1] + "/"
				for i := 2; i < len(nameSplited); i++ {
					modifiedFilePath += nameSplited[i]
					if i != len(nameSplited)-1 {
						modifiedFilePath += "_"
					}
				}
				//For Conv Param
				if nameSplited[2] == "conv1" || nameSplited[2] == "conv2" {
					var x int
					if nameSplited[2] == "conv1" {
						x = 1
					} else {
						x = 2
					}
					makeModifyMulParKernel(originalFolderPath+originalFileName, modifiedFilePath, "CONV2", originalFolderPath+nameSplited[0]+"_"+nameSplited[1]+"_bn"+strconv.Itoa(x))
					//For BN param
				} else if (nameSplited[2] == "bn1" && nameSplited[3] == "bias.txt") || (nameSplited[2] == "bn2" && nameSplited[3] == "bias.txt") {
					makeBn(32, 1, originalFolderPath+nameSplited[0]+"_"+nameSplited[1]+"_"+nameSplited[2], modifiedFolderPath+nameSplited[0]+"/"+nameSplited[1]+"/"+nameSplited[2])
				}
			case "layer2":
				//Make modified file path
				modifiedFilePath = modifiedFilePath + "layer2/" + nameSplited[1] + "/"
				for i := 2; i < len(nameSplited); i++ {
					modifiedFilePath += nameSplited[i]
					if i != len(nameSplited)-1 {
						modifiedFilePath += "_"
					}
				}
				//For Conv Param
				if nameSplited[2] == "conv1" || nameSplited[2] == "conv2" {
					var x int
					if nameSplited[2] == "conv1" {
						x = 1
					} else {
						x = 2
					}
					if nameSplited[1] == "0" && nameSplited[2] == "conv1" {
						makeModifyMulParKernel(originalFolderPath+originalFileName, modifiedFilePath, "CONV3s2", originalFolderPath+nameSplited[0]+"_"+nameSplited[1]+"_bn"+strconv.Itoa(x))
					} else {
						makeModifyMulParKernel(originalFolderPath+originalFileName, modifiedFilePath, "CONV3", originalFolderPath+nameSplited[0]+"_"+nameSplited[1]+"_bn"+strconv.Itoa(x))
					}
					//For BN param
				} else if (nameSplited[2] == "bn1" && nameSplited[3] == "bias.txt") || (nameSplited[2] == "bn2" && nameSplited[3] == "bias.txt") {
					makeBn(16, 2, originalFolderPath+nameSplited[0]+"_"+nameSplited[1]+"_"+nameSplited[2], modifiedFolderPath+nameSplited[0]+"/"+nameSplited[1]+"/"+nameSplited[2])
				}

			case "layer3":
				//Make modified file path
				modifiedFilePath = modifiedFilePath + "layer3/" + nameSplited[1] + "/"
				for i := 2; i < len(nameSplited); i++ {
					modifiedFilePath += nameSplited[i]
					if i != len(nameSplited)-1 {
						modifiedFilePath += "_"
					}
				}
				//For Conv Param
				if nameSplited[2] == "conv1" || nameSplited[2] == "conv2" {
					var x int
					if nameSplited[2] == "conv1" {
						x = 1
					} else {
						x = 2
					}
					if nameSplited[1] == "0" && nameSplited[2] == "conv1" {
						makeModifyMulParKernel(originalFolderPath+originalFileName, modifiedFilePath, "CONV4s2", originalFolderPath+nameSplited[0]+"_"+nameSplited[1]+"_bn"+strconv.Itoa(x))
					} else {
						makeModifyMulParKernel(originalFolderPath+originalFileName, modifiedFilePath, "CONV4", originalFolderPath+nameSplited[0]+"_"+nameSplited[1]+"_bn"+strconv.Itoa(x))
					}
					//For BN param
				} else if (nameSplited[2] == "bn1" && nameSplited[3] == "bias.txt") || (nameSplited[2] == "bn2" && nameSplited[3] == "bias.txt") {
					makeBn(8, 4, originalFolderPath+nameSplited[0]+"_"+nameSplited[1]+"_"+nameSplited[2], modifiedFolderPath+nameSplited[0]+"/"+nameSplited[1]+"/"+nameSplited[2])
				}
			case "linear":
				// modifiedFilePath = modifiedFilePath + "linear/" + nameSplited[1]
				// if nameSplited[1] == "bias.txt" {
				// 	makeBias(originalFolderPath+originalFileName, modifiedFilePath)
				// } else if nameSplited[1] == "weight.txt" {
				// 	makeLinearWeight(originalFolderPath+originalFileName, modifiedFilePath)
				// }
			default:
				//Make modified file path
				modifiedFilePath += "layer0/0/"
				//For Conv Param
				if nameSplited[0] == "conv1" {
					modifiedFilePath += "conv1_weight.txt" //CONV1이라 특이하게 적용.
					makeModifyMulParKernel(originalFolderPath+originalFileName, modifiedFilePath, "CONV1", originalFolderPath+"bn1")

					//For BN1 param
				} else if nameSplited[0] == "bn1" && nameSplited[1] == "bias.txt" {
					makeBn(32, 1, originalFolderPath+nameSplited[0], modifiedFilePath+"bn1")
				}
			}

			return nil
		})

		if err != nil {
			fmt.Println("오류:", err)
		}
	}

}
func makeModifyMulParKernel(inputFilePath, outputFilePath, convID, inputBNPath string) {
	originalKernel := kernelTxtToVector(inputFilePath)

	mapFeatures := GetMulParConvFeature(convID)

	thick := [][]int{
		{0, 1, 0, 1}, {0, 1, 0, 0}, {0, 1, 1, 0}, // 1, 2, 3
		{0, 0, 0, 1}, {0, 0, 0, 0}, {0, 0, 1, 0}, // 4, 5, 6
		{1, 0, 0, 1}, {1, 0, 0, 0}, {1, 0, 1, 0}, // 7, 8, 9
	}

	var flattenFilter [][]float64
	var rotateNums []int

	for height := -1; height < 2; height++ {
		for width := -1; width < 2; width++ {
			rotateNums = append(rotateNums, mapFeatures.InputDataWidth*height+width)
		}
	}

	for t := 0; t < 9; t++ {
		flattenFilter = append(flattenFilter, rotate(flatten(makeZeroBorderOnes(thick[t], int(mapFeatures.InputDataWidth))), rotateNums[t]))
	}

	runningVar := simpleTxtReader(inputBNPath + "_running_var.txt")
	weight := simpleTxtReader(inputBNPath + "_weight.txt")
	var bnMult []float64

	for i := 0; i < len(weight); i++ {
		bnMult = append(bnMult, weight[i]/math.Sqrt(runningVar[i]+0.00001))
	}

	var outputKernel [][][]float64
	for km := 0; km < len(mapFeatures.KernelMap); km++ {
		outputKernel = append(outputKernel, make([][]float64, 9))
		for w := 0; w < 9; w++ {
			outputKernel[km][w] = make([]float64, 0)
			for _, curKernel := range mapFeatures.KernelMap[km] {
				for c := 0; c < len(originalKernel[0]); c++ {
					for _, x := range multVecAndConst(flattenFilter[w], originalKernel[curKernel][c][w/3][w%3]) {
						outputKernel[km][w] = append(outputKernel[km][w], x*bnMult[curKernel])
					}
				}
				if convID == "CONV1" {
					for x := 0; x < 1024; x++ {
						outputKernel[km][w] = append(outputKernel[km][w], 0)
					}
				}
			}
			// fmt.Println(len(mapFeatures.kernelMap[km]), len(originalKernel[0]), len(flattenFilter[w]), len(outputKernel[km][w]))
		}
	}

	for km := range outputKernel {
		for w := range outputKernel[km] {
			outputKernel[km][w] = multiplex(outputKernel[km][w], mapFeatures.InputDataWidth, mapFeatures.K)
		}
	}

	outputFilePath = outputFilePath[:len(outputFilePath)-4]
	for km := 0; km < len(outputKernel); km++ {
		for w := 0; w < len(outputKernel[0]); w++ {
			tempFilePath := outputFilePath + strconv.Itoa(km) + "_" + strconv.Itoa(w) + ".txt"
			//open
			outputFile, err := os.Create(tempFilePath)
			if err != nil {
				fmt.Println("Failed to open file:", err)
				return
			}
			defer outputFile.Close()
			//write
			writer := bufio.NewWriter(outputFile)
			for _, value := range outputKernel[km][w] {
				_, err := fmt.Fprintf(writer, "%.15f\n", value)
				if err != nil {
					fmt.Fprintf(os.Stderr, "modifyKernel를 파일에 쓰는 도중 오류가 발생했습니다.: %v\n", err)
					return
				}
			}
			if err := writer.Flush(); err != nil {
				fmt.Fprintf(os.Stderr, "modifyKernel를 파일에 쓰는 도중 오류가 발생했습니다.: %v\n", err)
				return
			}

			fmt.Printf("%s : modifyKernel이 저장되었습니다. 길이: %d\n", tempFilePath, len(outputKernel[0][0]))
		}
	}

}

// Making bn_add. Make bnMult too, but don't save.
func makeBn(dataWidth, packing int, inputBNPath, outputBNPath string) {
	// 변수 설정
	var bias, runningMean, runningVar, weight, alpha, bnAdd []float64 //bnMult

	// 변수 가져오기
	bias = simpleTxtReader(inputBNPath + "_bias.txt")
	runningMean = simpleTxtReader(inputBNPath + "_running_mean.txt")
	runningVar = simpleTxtReader(inputBNPath + "_running_var.txt")
	weight = simpleTxtReader(inputBNPath + "_weight.txt")

	// 출력 파일 만들기
	// ((x-mean)/root(var+0.00001))*weight+bias => x*alpha + (bias-mean*alpha)
	// (alpha = weight/root(var+0.00001))
	for i := 0; i < len(weight); i++ {
		alpha = append(alpha, weight[i]/math.Sqrt(runningVar[i]+0.00001))
	}

	// bnMult = alpha

	for i := 0; i < len(weight); i++ {
		bnAdd = append(bnAdd, bias[i]-runningMean[i]*alpha[i])
	}

	// 패킹
	bnAdd = packAndCopy(dataWidth, packing, bnAdd)
	// bnMult = packAndCopy(dataWidth, packing, bnMult)

	// 결과 저장

	// 디렉터리 생성
	if err := os.MkdirAll(outputBNPath[:len(outputBNPath)-3], 0755); err != nil {
		fmt.Fprintf(os.Stderr, "디렉터리를 생성할 수 없습니다.: %s\n", outputBNPath)
		return
	}
	// bn_add 저장
	outputFile, err := os.Create(outputBNPath + "_add.txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "수정된 bn_add를 저장할 파일을 열 수 없습니다.: %s\n", outputBNPath+"_add.txt")
		return
	}
	defer outputFile.Close()

	writer := bufio.NewWriter(outputFile)
	for _, value := range bnAdd {
		_, err := fmt.Fprintf(writer, "%.15f\n", value)
		if err != nil {
			fmt.Fprintf(os.Stderr, "수정된 bn_add를 파일에 쓰는 도중 오류가 발생했습니다.: %v\n", err)
			return
		}
	}
	if err := writer.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "수정된 bn_add를 파일에 쓰는 도중 오류가 발생했습니다.: %v\n", err)
		return
	}

	fmt.Printf("%s : 수정된 bn_add가 저장되었습니다. 길이: %d\n", outputBNPath+"_add.txt", len(bnAdd))
}

func makeModifyKernel(inputFilePath, outputFilePath, convID, inputBNPath string) {
	originalKernel := kernelTxtToVector(inputFilePath)

	mapFeatures := GetConvFeature(convID)

	thick := [][]int{
		{0, 1, 0, 1}, {0, 1, 0, 0}, {0, 1, 1, 0}, // 1, 2, 3
		{0, 0, 0, 1}, {0, 0, 0, 0}, {0, 0, 1, 0}, // 4, 5, 6
		{1, 0, 0, 1}, {1, 0, 0, 0}, {1, 0, 1, 0}, // 7, 8, 9
	}

	var flattenFilter [][]float64
	var rotateNums []int

	for height := -1; height < 2; height++ {
		for width := -1; width < 2; width++ {
			rotateNums = append(rotateNums, mapFeatures.InputDataWidth*height+width)
		}
	}

	for t := 0; t < 9; t++ {
		flattenFilter = append(flattenFilter, rotate(flatten(makeZeroBorderOnes(thick[t], int(mapFeatures.InputDataWidth))), rotateNums[t]))
	}

	runningVar := simpleTxtReader(inputBNPath + "_running_var.txt")
	weight := simpleTxtReader(inputBNPath + "_weight.txt")
	var bnMult []float64

	for i := 0; i < len(weight); i++ {
		bnMult = append(bnMult, weight[i]/math.Sqrt(runningVar[i]+0.00001))
	}

	var outputKernel [][][]float64
	for km := 0; km < len(mapFeatures.KernelMap); km++ {
		outputKernel = append(outputKernel, make([][]float64, 9))
		for w := 0; w < 9; w++ {
			outputKernel[km][w] = make([]float64, 0)
			for _, curKernel := range mapFeatures.KernelMap[km] {
				for c := 0; c < len(originalKernel[0]); c++ {
					for _, x := range multVecAndConst(flattenFilter[w], originalKernel[curKernel][c][w/3][w%3]) {
						outputKernel[km][w] = append(outputKernel[km][w], x*bnMult[curKernel])
					}
				}
				if convID == "CONV1" {
					for x := 0; x < 1024; x++ {
						outputKernel[km][w] = append(outputKernel[km][w], 0)
					}
				}
			}
			// fmt.Println(len(mapFeatures.kernelMap[km]), len(originalKernel[0]), len(flattenFilter[w]), len(outputKernel[km][w]))
		}
	}

	for km := range outputKernel {
		for w := range outputKernel[km] {
			outputKernel[km][w] = multiplex(outputKernel[km][w], mapFeatures.InputDataWidth, mapFeatures.K)
		}
	}

	outputFilePath = outputFilePath[:len(outputFilePath)-4]
	for km := 0; km < len(outputKernel); km++ {
		for w := 0; w < len(outputKernel[0]); w++ {
			tempFilePath := outputFilePath + strconv.Itoa(km) + "_" + strconv.Itoa(w) + ".txt"
			//open
			outputFile, err := os.Create(tempFilePath)
			if err != nil {
				fmt.Println("Failed to open file:", err)
				return
			}
			defer outputFile.Close()
			//write
			writer := bufio.NewWriter(outputFile)
			for _, value := range outputKernel[km][w] {
				_, err := fmt.Fprintf(writer, "%.15f\n", value)
				if err != nil {
					fmt.Fprintf(os.Stderr, "modifyKernel를 파일에 쓰는 도중 오류가 발생했습니다.: %v\n", err)
					return
				}
			}
			if err := writer.Flush(); err != nil {
				fmt.Fprintf(os.Stderr, "modifyKernel를 파일에 쓰는 도중 오류가 발생했습니다.: %v\n", err)
				return
			}

			fmt.Printf("%s : modifyKernel이 저장되었습니다. 길이: %d\n", tempFilePath, len(outputKernel[0][0]))
		}
	}

}

func packAndCopy(dataWidth, packing int, inputVector []float64) []float64 {
	var beforePack []float64
	var afterPack []float64

	for oneData := 0; oneData < 32768; oneData += len(inputVector) * dataWidth * dataWidth {
		for _, value := range inputVector {
			for i := 0; i < dataWidth; i++ {
				for j := 0; j < dataWidth; j++ {
					beforePack = append(beforePack, value)
				}
			}
		}
	}

	if packing == 1 {
		return beforePack
	} else {
		afterPack = multiplex(beforePack, dataWidth, packing)
		return afterPack
	}
}
func kernelTxtToVector(inputFilePath string) [][][][]float64 {
	file, err := os.Open(inputFilePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "파일을 열 수 없습니다.")
		os.Exit(1)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Scan()
	line := scanner.Text()

	kernelShape := splitWithSpace(line)
	kernelSize := int(kernelShape[2])
	channel := int(kernelShape[1])
	kernelNum := int(kernelShape[0])

	kernelWeight := make([][][][]float64, kernelNum)
	for kn := 0; kn < kernelNum; kn++ {
		kernelWeight[kn] = make([][][]float64, channel)
		for c := 0; c < channel; c++ {
			kernelWeight[kn][c] = make([][]float64, kernelSize)
			for ks1 := 0; ks1 < kernelSize; ks1++ {
				kernelWeight[kn][c][ks1] = make([]float64, kernelSize)
				for ks2 := 0; ks2 < kernelSize; ks2++ {
					for scanner.Scan() {
						line = scanner.Text()
						if line != "" {
							break
						}
					}
					value, err := strconv.ParseFloat(line, 64)
					if err != nil {
						fmt.Fprintln(os.Stderr, "숫자 변환 오류:", err)
						os.Exit(1)
					}
					kernelWeight[kn][c][ks1][ks2] = value
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "파일 읽기 오류:", err)
		os.Exit(1)
	}

	return kernelWeight
}
func splitWithSpace(str string) []float64 {
	strNumbers := strings.Fields(str)
	doubleNumbers := make([]float64, 0, len(strNumbers))

	for _, s := range strNumbers {
		num, err := strconv.ParseFloat(s, 64)
		if err != nil {
			fmt.Println("Error:", err)
			continue
		}
		doubleNumbers = append(doubleNumbers, num)
	}

	return doubleNumbers
}
func flatten(input [][]float64) []float64 {
	var result []float64

	for yy := 0; yy < len(input); yy++ {
		for xx := 0; xx < len(input[0]); xx++ {
			result = append(result, input[yy][xx])
		}
	}

	return result
}

func rotate(input []float64, rotateNum int) []float64 {
	size := len(input)
	result := make([]float64, 0)

	if rotateNum < 0 {
		rotateNum += size
	}

	for s := rotateNum; s < size; s++ {
		result = append(result, input[s])
	}
	for s := 0; s < rotateNum; s++ {
		result = append(result, input[s])
	}

	return result
}

func makeZeroBorderOnes(UDLR []int, sideSize int) [][]float64 {
	ones := make([][]float64, sideSize)
	for i := range ones {
		ones[i] = make([]float64, sideSize)
		for j := range ones[i] {
			ones[i][j] = 1
		}
	}

	// UP
	for up := 0; up < UDLR[0]; up++ {
		for width := 0; width < sideSize; width++ {
			ones[up][width] = 0
		}
	}

	// DOWN
	for down := sideSize - 1; down > sideSize-1-UDLR[1]; down-- {
		for width := 0; width < sideSize; width++ {
			ones[down][width] = 0
		}
	}

	// LEFT
	for left := 0; left < UDLR[2]; left++ {
		for height := 0; height < sideSize; height++ {
			ones[height][left] = 0
		}
	}

	// RIGHT
	for right := sideSize - 1; right > sideSize-1-UDLR[3]; right-- {
		for height := 0; height < sideSize; height++ {
			ones[height][right] = 0
		}
	}

	return ones
}

func simpleTxtReader(inputFilePath string) []float64 {
	file, err := os.Open(inputFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "파일을 열 수 없습니다.: %s\n", inputFilePath)
		return nil
	}
	defer file.Close()

	returnVector := make([]float64, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			break
		}
		num, err := strconv.ParseFloat(line, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "숫자로 변환할 수 없습니다.: %s\n", line)
			continue
		}
		returnVector = append(returnVector, num)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "파일을 읽는 도중 오류가 발생했습니다.: %v\n", err)
		return nil
	}

	return returnVector
}

func multVecAndConst(A []float64, B float64) []float64 {
	var result []float64
	for x := 0; x < len(A); x++ {
		result = append(result, A[x]*B)
	}
	return result
}

func multVec(A []float64, B []float64) []float64 {
	var result []float64
	for x := 0; x < len(A); x++ {
		result = append(result, A[x]*B[x])
	}
	return result
}

func addVec(A []float64, B []float64) []float64 {
	var result []float64
	for x := 0; x < len(A); x++ {
		result = append(result, A[x]+B[x])
	}
	return result
}

// OR operation for A and B
func AndVec(A []float64, B []float64) []float64 {
	var result []float64
	for x := 0; x < len(A); x++ {
		if A[x] == 1 || B[x] == 1 {
			result = append(result, 1)
		} else {
			result = append(result, 0)
		}

	}
	return result
}

func multiplex(input []float64, dataWidth, k int) []float64 {
	if k == 1 {
		return input
	}

	beforeChannel := len(input) / (dataWidth * dataWidth)

	var input3d [][][]float64
	for z := 0; z < beforeChannel; z++ {
		var temp [][]float64
		for y := 0; y < dataWidth; y++ {
			temp = append(temp, make([]float64, dataWidth))
		}
		input3d = append(input3d, temp)
	}

	var output3d [][][]float64
	for zz := 0; zz < beforeChannel/k/k; zz++ {
		var temp [][]float64
		for yy := 0; yy < dataWidth*k; yy++ {
			temp = append(temp, make([]float64, dataWidth*k))
		}
		output3d = append(output3d, temp)
	}

	index := 0
	for z := 0; z < beforeChannel; z++ {
		for y := 0; y < dataWidth; y++ {
			for x := 0; x < dataWidth; x++ {
				input3d[z][y][x] = input[index]
				index++
			}
		}
	}

	for zz := 0; zz < beforeChannel; zz++ {
		for yy := 0; yy < dataWidth; yy++ {
			for xx := 0; xx < dataWidth; xx++ {
				output3d[zz/k/k][yy*k+((zz%(k*k))/k)][xx*k+((zz%(k*k))%k)] = input3d[zz][yy][xx]
			}
		}
	}

	var result []float64
	for _, z := range output3d {
		for _, y := range z {
			for _, x := range y {
				result = append(result, x)
			}
		}
	}

	return result
}

func kernelTxtToSlice(inputFilePath string) [][][][]float64 {
	file, err := os.Open(inputFilePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "파일을 열 수 없습니다.")
		os.Exit(1)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Scan()
	line := scanner.Text()

	kernelShape := splitStringToFloats(line)
	kernelSize := int(kernelShape[2])
	channel := int(kernelShape[1])
	kernelNum := int(kernelShape[0])

	kernelWeight := make([][][][]float64, kernelNum)
	for kn := 0; kn < kernelNum; kn++ {
		kernelWeight[kn] = make([][][]float64, channel)
		for c := 0; c < channel; c++ {
			kernelWeight[kn][c] = make([][]float64, kernelSize)
			for ks1 := 0; ks1 < kernelSize; ks1++ {
				kernelWeight[kn][c][ks1] = make([]float64, kernelSize)
				for ks2 := 0; ks2 < kernelSize; ks2++ {
					for scanner.Scan() {
						line = scanner.Text()
						if line != "" {
							break
						}
					}
					value, err := strconv.ParseFloat(line, 64)
					if err != nil {
						fmt.Fprintln(os.Stderr, "숫자 변환 오류:", err)
						os.Exit(1)
					}
					kernelWeight[kn][c][ks1][ks2] = value
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "파일 읽기 오류:", err)
		os.Exit(1)
	}

	return kernelWeight
}

func splitStringToFloats(input string) []float64 {
	fields := strings.Fields(input)
	result := make([]float64, len(fields))

	for i, field := range fields {
		value, err := strconv.ParseFloat(field, 64)
		if err != nil {
			return nil
		}
		result[i] = value
	}

	return result
}

func print4DArray(arr [][][][]float64) {
	for i := 0; i < len(arr); i++ {
		fmt.Printf("Layer %d:\n", i+1)
		for j := 0; j < len(arr[i]); j++ {
			fmt.Printf("  Sublayer %d:\n", j+1)
			for k := 0; k < len(arr[i][j]); k++ {
				for l := 0; l < len(arr[i][j][k]); l++ {
					fmt.Printf("%v ", arr[i][j][k][l])
				}
				fmt.Println()
			}
			fmt.Println()
		}
		fmt.Println()
	}
}
