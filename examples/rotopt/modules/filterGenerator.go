package modules

import (
	"bufio"
	"fmt"
	"math"
	"os"
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

	for h := 0; h < 32; h++ {
		for w := 0; w < 32; w++ {
			if w%k == 0 && h%k == 0 {
				result[w+h*32] = 1
				// fmt.Println(w + h*32)
			}
		}
	}

	channelRot := -(1024*(channel/(k*k)) + 32*(channel%(k*k)/k) + (channel % (k * k) % k))
	result = PlainRot(result, channelRot)
	// fmt.Println("c : ", channelRot)

	parRot := -(32768 / allPar * parNum)
	result = PlainRot(result, parRot)
	// fmt.Println("p : ", parRot)
	return result
}

func MakeTxtRotOptConvFilter(convID string, depth int, encoder *ckks.Encoder, params ckks.Parameters) (preCompFilter [][]*rlwe.Plaintext, lastFilter [][]*rlwe.Plaintext) {

	//get ConvMap
	convMap, _, _ := GetConvBlueprints(convID, depth)

	// get convFeature
	convFeatureMap := GetRotOptConvFeature(convID)

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

			for zeroCheck := treeDepth - 1; zeroCheck > 0; zeroCheck-- {
				if convMap[zeroCheck][0] == 0 {
					for t := 0; t < len(nonSplitFilter); t++ {
						nonSplitFilter[t] = multVec(nonSplitFilter[t], mode0Filter(convMap[zeroCheck][1]))
					}
				} else {
					break
				}
			}

			if convFeatureMap.Stride != 1 {
				for t := 0; t < len(nonSplitFilter); t++ {
					nonSplitFilter[t] = multVec(nonSplitFilter[t], StrideFilter(convFeatureMap.K, convFeatureMap.InputDataWidth))
				}
			}
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
func StrideFilter(k int, w int) []float64 { //for stride=2
	var strideFilter []float64

	if k == 1 {
		for ii := 0; ii < 32768/(2*w*k); ii++ {
			for i := 0; i < w*k/2; i++ {
				strideFilter = append(strideFilter, 1, 0)
			}
			for i := 0; i < w*k; i++ {
				strideFilter = append(strideFilter, 0)
			}
		}
	} else if k == 2 {
		for ii := 0; ii < 32768/(4*w*k); ii++ {
			for i := 0; i < w*k*2/4; i++ {
				strideFilter = append(strideFilter, 1, 1, 0, 0)
			}
			for i := 0; i < w*k*2; i++ {
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
			if (i/rotIndex)%2 == 0 {
				resultFilter = append(resultFilter, 1)
			} else {
				resultFilter = append(resultFilter, 0)
			}
		}
	} else {
		for i := 0; i < 32768; i++ {
			if (i/rotIndex)%2 == 0 {
				resultFilter = append(resultFilter, 0)
			} else {
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
func LeftUpFilter(cf *ConvFeature) []float64 {
	var filter []float64
	k := cf.K
	for block := 0; block < cf.BeforeCopy; block++ {
		for i := 0; i < 32768/cf.BeforeCopy; i++ {
			if i%k == 0 && (i/(cf.InputDataWidth*k))%k == 0 && i < cf.InputDataWidth*cf.InputDataHeight*k*k {
				filter = append(filter, 1)
			} else {
				filter = append(filter, 0)
			}
		}
	}

	return filter
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
	for km := 0; km < len(mapFeatures.KernelBP); km++ {
		outputKernel = append(outputKernel, make([][]float64, 9))
		for w := 0; w < 9; w++ {
			outputKernel[km][w] = make([]float64, 0)
			for _, curKernel := range mapFeatures.KernelBP[km] {
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
				if convID == "CvTCifar100Stage3" {
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
					fmt.Fprintf(os.Stderr, "modifyKernel write error: %v\n", err)
					return
				}
			}
			if err := writer.Flush(); err != nil {
				fmt.Fprintf(os.Stderr, "modifyKernel write error: %v\n", err)
				return
			}

			fmt.Printf("%s : modifyKernel saved. length : %d\n", tempFilePath, len(outputKernel[0][0]))
		}
	}

}

func makeModifyKernel(inputFilePath, outputFilePath, convID, inputBNPath string) {
	originalKernel := kernelTxtToVector(inputFilePath)

	mapFeatures := GetRotOptConvFeature(convID)

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
	for km := 0; km < len(mapFeatures.KernelBP); km++ {
		outputKernel = append(outputKernel, make([][]float64, 9))
		for w := 0; w < 9; w++ {
			outputKernel[km][w] = make([]float64, 0)
			for _, curKernel := range mapFeatures.KernelBP[km] {
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
				if convID == "CvTCifar100Stage3" {
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
					fmt.Fprintf(os.Stderr, "modifyKernel write error: %v\n", err)
					return
				}
			}
			if err := writer.Flush(); err != nil {
				fmt.Fprintf(os.Stderr, "modifyKernel write error:: %v\n", err)
				return
			}

			fmt.Printf("%s : modifyKernel write success. length: %d\n", tempFilePath, len(outputKernel[0][0]))
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
		fmt.Fprintln(os.Stderr, "cannot open file.")
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
						fmt.Fprintln(os.Stderr, "number trans error:", err)
						os.Exit(1)
					}
					kernelWeight[kn][c][ks1][ks2] = value
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "file read error:", err)
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

func add(input []float64, input2 []float64) []float64 {
	size := len(input)
	result := make([]float64, 0)
	if len(input) != len(input2) {
		fmt.Println("Size is different in add()!")
	}
	for index := 0; index < size; index++ {
		result = append(result, input[index]+input2[index])
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
		fmt.Fprintf(os.Stderr, "cannot open file.: %s\n", inputFilePath)
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
			fmt.Fprintf(os.Stderr, "cannot trans numbers.: %s\n", line)
			continue
		}
		returnVector = append(returnVector, num)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "error during reading: %v\n", err)
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
		fmt.Fprintln(os.Stderr, "cannot open file.")
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
						fmt.Fprintln(os.Stderr, "number trans error:", err)
						os.Exit(1)
					}
					kernelWeight[kn][c][ks1][ks2] = value
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "file read error:", err)
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
