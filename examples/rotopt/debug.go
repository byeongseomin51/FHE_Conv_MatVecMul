package main

import (
	"bufio"
	"fmt"
	"math"
	"math/big"
	"math/bits"
	"math/rand"
	"os"
	"rotopt/modules"
	"strconv"
	"time"

	"github.com/tuneinsight/lattigo/v5/core/rlwe"
	"github.com/tuneinsight/lattigo/v5/schemes/ckks"
	"github.com/tuneinsight/lattigo/v5/utils/sampling"
)

func int2dTo1d(input [][]int) []int {
	var result []int
	for _, i := range input {
		for _, j := range i {
			result = append(result, j)
		}
	}
	return result
}
func zeroFilter(input float64) float64 {
	if input > 0.00001 || input < -0.00001 {
		return input
	} else {
		return 0
	}
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

// 유클리디안 거리를 계산하는 함수
func euclideanDistance(arr1, arr2 []float64) float64 {
	if len(arr1) != len(arr2) {
		panic("두 배열의 길이는 동일해야 합니다." + strconv.Itoa(len(arr1)) + "," + strconv.Itoa(len(arr2)))
	}

	var distance float64

	for i := 0; i < len(arr1); i++ {
		distance += math.Pow(arr1[i]-arr2[i], 2)

		// if math.Abs(arr1[i]-arr2[i]) > 0.001 {
		// 	fmt.Print(i, " ")
		// }
	}

	return math.Sqrt(distance)
}
func makeRandomPlain(length int, encoder ckks.Encoder, pt *rlwe.Plaintext) {
	floats := makeRandomFloat(length)

	encoder.Encode(floats, pt)

}
func floatToTxt(filePath string, floats []float64) {

	if _, err := os.Stat(filePath); os.IsNotExist(err) {

		file, err := os.Create(filePath)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer file.Close()

		for _, val := range floats {

			_, err := file.WriteString(fmt.Sprintf("%.15f\n", val))
			if err != nil {
				fmt.Println(err)
				return
			}
		}

		fmt.Printf("File '%s' created successfully.\n", filePath)
	} else {

		file, err := os.Create(filePath)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer file.Close()

		for _, val := range floats {

			_, err := file.WriteString(fmt.Sprintf("%.15f\n", val))
			if err != nil {
				fmt.Println(err)
				return
			}
		}
		fmt.Printf("File '%s' already exists. Overwrited\n", filePath)
	}

}
func makeRandomFloat(length int) []float64 {
	valuesWant := make([]float64, length)
	for i := range valuesWant {
		valuesWant[i] = sampling.RandFloat64(-1, 1)
	}
	return valuesWant
}
func makeRandomInput(channel, height, width int) [][][]float64 {
	randomVector := make([][][]float64, channel)
	for d := 0; d < channel; d++ {
		randomVector[d] = make([][]float64, height)
		for i := 0; i < height; i++ {
			randomVector[d][i] = make([]float64, width)
			for j := 0; j < width; j++ {
				randomVector[d][i][j] = sampling.RandFloat64(-1, 1)
			}
		}
	}

	return randomVector
}

func makeRandomKernel(outChannel, inChannel, kernelHeight, kernelWidth int) [][][][]float64 {
	kernel := make([][][][]float64, outChannel)
	for o := 0; o < outChannel; o++ {
		kernel[o] = make([][][]float64, inChannel)
		for i := 0; i < inChannel; i++ {
			kernel[o][i] = make([][]float64, kernelHeight)
			for h := 0; h < kernelHeight; h++ {
				kernel[o][i][h] = make([]float64, kernelWidth)
				for w := 0; w < kernelWidth; w++ {
					kernel[o][i][h][w] = sampling.RandFloat64(-1, 1)
				}
			}
		}
	}
	return kernel
}

func convertComplexToFloat(slice []complex128) []float64 {
	floatSlice := make([]float64, len(slice))

	for i, v := range slice {
		floatSlice[i] = real(v)
	}

	return floatSlice
}

func generateRandomComplexArray(length int) []float64 {

	arr := make([]float64, length)

	for i := 0; i < length; i++ {
		arr[i] = rand.Float64()
	}

	return arr
}
func generateRandomFloatArray(length int) []float64 {

	arr := make([]float64, length)

	for i := 0; i < length; i++ {
		arr[i] = rand.Float64()
	}

	return arr
}
func generateRandomBigFloatArray(length int) []*big.Float {

	arr := make([]*big.Float, length)

	for i := 0; i < length; i++ {
		randomFloat := new(big.Float).SetFloat64(rand.Float64())
		arr[i] = randomFloat
	}

	return arr
}
func generateEmptyBigFloatArray(length int) []*big.Float {

	arr := make([]*big.Float, length)
	zero := new(big.Float).SetFloat64(0.0)
	for i := 0; i < length; i++ {
		arr[i] = new(big.Float).Set(zero)
	}

	return arr
}

// ZeroPad pads the input tensor with zeros
func ZeroPad(input [][][]float64, pad int) [][][]float64 {
	inChannels := len(input)
	inHeight := len(input[0])
	inWidth := len(input[0][0])
	paddedHeight := inHeight + 2*pad
	paddedWidth := inWidth + 2*pad

	padded := make([][][]float64, inChannels)
	for c := 0; c < inChannels; c++ {
		padded[c] = make([][]float64, paddedHeight)
		for i := 0; i < paddedHeight; i++ {
			padded[c][i] = make([]float64, paddedWidth)
			for j := 0; j < paddedWidth; j++ {
				// Center part is the original input
				if i >= pad && i < pad+inHeight && j >= pad && j < pad+inWidth {
					padded[c][i][j] = input[c][i-pad][j-pad]
				} else {
					padded[c][i][j] = 0.0
				}
			}
		}
	}
	return padded
}

// Conv2d with stride and padding
func PlainConvolution2D(input [][][]float64, kernel [][][][]float64, stride, pad int) [][][]float64 {
	inChannels := len(input)
	inHeight := len(input[0])
	inWidth := len(input[0][0])

	outChannels := len(kernel)
	kernelHeight := len(kernel[0][0])
	kernelWidth := len(kernel[0][0][0])

	// Apply padding
	paddedInput := ZeroPad(input, pad)
	paddedHeight := inHeight + 2*pad
	paddedWidth := inWidth + 2*pad

	// Output size
	outHeight := (paddedHeight-kernelHeight)/stride + 1
	outWidth := (paddedWidth-kernelWidth)/stride + 1

	// Initialize output
	output := make([][][]float64, outChannels)
	for oc := 0; oc < outChannels; oc++ {
		output[oc] = make([][]float64, outHeight)
		for i := 0; i < outHeight; i++ {
			output[oc][i] = make([]float64, outWidth)
		}
	}

	// Convolution
	for oc := 0; oc < outChannels; oc++ {
		for i := 0; i < outHeight; i++ {
			for j := 0; j < outWidth; j++ {
				sum := 0.0
				for ic := 0; ic < inChannels; ic++ {
					for kh := 0; kh < kernelHeight; kh++ {
						for kw := 0; kw < kernelWidth; kw++ {
							h := i*stride + kh
							w := j*stride + kw
							sum += paddedInput[ic][h][w] * kernel[oc][ic][kh][kw]
						}
					}
				}
				output[oc][i][j] = sum
			}
		}
	}
	return output
}

func sample1DComplex(arr []complex128, start int, end int) {
	for i := start; i < end; i++ {
		fmt.Printf("%v ", arr[i])
	}
	fmt.Println()
}
func msgSample1DArray(msg string, arr []float64, start int, end int) {
	fmt.Println(msg)
	for i := start; i < end; i++ {
		fmt.Printf("%v ", arr[i])
	}
	fmt.Println()
}
func sample1DArray(arr []float64, start int, end int) {
	for i := start; i < end; i++ {
		if arr[i] > 0.00001 || arr[i] < -0.00001 {
			fmt.Printf("%v ", arr[i])
		} else {
			fmt.Print("0 ")
		}

	}
	fmt.Println()
}
func print1DArray(arr []float64) {
	for i := 0; i < len(arr); i++ {
		if arr[i] < 0.00001 && arr[i] > -0.00001 {
			fmt.Printf("%v ", arr[i])
		} else {
			fmt.Print("0 ")
		}

	}
	fmt.Println()
}

func print2DArray(arr [][]float64) {
	for i := 0; i < len(arr); i++ {
		for j := 0; j < len(arr[i]); j++ {
			fmt.Printf("%v ", arr[i][j])
		}
		fmt.Println()
	}
}

func print3DArray(arr [][][]float64) {
	for i := 0; i < len(arr); i++ {
		fmt.Printf("Layer %d:\n", i+1)
		for j := 0; j < len(arr[i]); j++ {
			for k := 0; k < len(arr[i][j]); k++ {
				fmt.Printf("%v ", arr[i][j][k])
			}
			fmt.Println()
		}
		fmt.Println()
	}
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
func flatten(input [][][]float64) []float64 {
	var result []float64
	for c := 0; c < len(input); c++ {
		for h := 0; h < len(input[0]); h++ {
			for w := 0; w < len(input[0][0]); w++ {
				result = append(result, input[c][h][w])
			}
		}
	}
	return result
}
func flatten2d(input [][]float64) []float64 {
	var result []float64

	for yy := 0; yy < len(input); yy++ {
		for xx := 0; xx < len(input[0]); xx++ {
			result = append(result, input[yy][xx])
		}
	}

	return result
}

// Multiplexed-parllel packed the input
func MulParPacking(input3d [][][]float64, cf *modules.ConvFeature, cc *customContext) []float64 {
	k := cf.K

	inputChannel := len(input3d)
	inputHeight := len(input3d[0])
	inputWidth := len(input3d[0][0])

	outputChannel := inputChannel / (k * k)
	outputHeight := inputHeight * k
	outputWidth := inputWidth * k

	// Initialize output3d
	output3d := make([][][]float64, outputChannel)
	for c := 0; c < outputChannel; c++ {
		output3d[c] = make([][]float64, outputHeight)
		for h := 0; h < outputHeight; h++ {
			output3d[c][h] = make([]float64, outputWidth)
		}
	}

	// Make plain output3d
	for c := 0; c < inputChannel; c++ {
		for h := 0; h < inputHeight; h++ {
			for w := 0; w < inputWidth; w++ {
				output3d[c/k/k][h*k+((c%(k*k))/k)][w*k+((c%(k*k))%k)] = input3d[c][h][w]
			}
		}
	}

	// Make it 1d
	var output1d []float64
	for _, c := range output3d {
		for _, h := range c {
			for _, w := range h {
				output1d = append(output1d, w)
			}
		}
	}
	if cf.ConvID == "CONV1" {
		for i := 0; i < 1024; i++ {
			output1d = append(output1d, 0)
		}
	}
	if cf.ConvID == "CvTCifar100Stage3" {
		for i := 0; i < 1024; i++ {
			output1d = append(output1d, 0)
		}
	}

	// Make it parallel
	cipherLen := 32768
	result := make([]float64, 0, cipherLen)
	for len(result) < cipherLen {
		remain := cipherLen - len(result)
		if remain >= len(output1d) {
			result = append(result, output1d...)
		} else {
			result = append(result, output1d[:remain]...)
		}
	}

	return result
}

// Encode the plain kernel
func EncodeKernel(kernel4d [][][][]float64, cf *modules.ConvFeature, cc *customContext) [][]*rlwe.Plaintext {

	thick := [][]int{
		{0, 1, 0, 1}, {0, 1, 0, 0}, {0, 1, 1, 0}, // 1, 2, 3
		{0, 0, 0, 1}, {0, 0, 0, 0}, {0, 0, 1, 0}, // 4, 5, 6
		{1, 0, 0, 1}, {1, 0, 0, 0}, {1, 0, 1, 0}, // 7, 8, 9
	}

	var flattenFilter [][]float64
	var rotateNums []int

	for height := -1; height < 2; height++ {
		for width := -1; width < 2; width++ {
			rotateNums = append(rotateNums, cf.InputDataWidth*height+width)
		}
	}

	for t := 0; t < 9; t++ {
		flattenFilter = append(flattenFilter, rotate(flatten2d(makeZeroBorderOnes(thick[t], int(cf.InputDataWidth))), rotateNums[t]))
	}

	var outputKernel [][][]float64
	for km := 0; km < len(cf.KernelBP); km++ {
		outputKernel = append(outputKernel, make([][]float64, 9))
		for w := 0; w < 9; w++ {
			outputKernel[km][w] = make([]float64, 0)
			for _, curKernel := range cf.KernelBP[km] {
				for c := 0; c < len(kernel4d[0]); c++ {
					for _, x := range multVecAndConst(flattenFilter[w], kernel4d[curKernel][c][w/3][w%3]) {
						outputKernel[km][w] = append(outputKernel[km][w], x)
					}
				}
				if cf.ConvID == "CONV1" {
					for x := 0; x < 1024; x++ {
						outputKernel[km][w] = append(outputKernel[km][w], 0)
					}
				}
				if cf.ConvID == "CvTCifar100Stage3" {
					for x := 0; x < 1024; x++ {
						outputKernel[km][w] = append(outputKernel[km][w], 0)
					}
				}
			}
		}
	}

	for km := range outputKernel {
		for w := range outputKernel[km] {
			outputKernel[km][w] = multiplex(outputKernel[km][w], cf.InputDataWidth, cf.K)
		}
	}

	//Encode to kernel
	var encodedKernel [][]*rlwe.Plaintext
	for km := 0; km < len(outputKernel); km++ {
		ws := make([]*rlwe.Plaintext, len(outputKernel[0]))
		for w := 0; w < len(outputKernel[0]); w++ {
			exPlain := ckks.NewPlaintext(cc.Params, cc.Params.MaxLevel())
			err := cc.Encoder.Encode(outputKernel[km][w], exPlain)
			if err != nil {
				fmt.Println(err)
			}
			ws[w] = exPlain
		}
		encodedKernel = append(encodedKernel, ws)
	}

	return encodedKernel
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

func multVecAndConst(A []float64, B float64) []float64 {
	var result []float64
	for x := 0; x < len(A); x++ {
		result = append(result, A[x]*B)
	}
	return result
}

// Decrypt UnMultiplexed-parllel packed the input
func UnMulParPacking(input *rlwe.Ciphertext, cf *modules.ConvFeature, cc *customContext) [][][]float64 {

	k := cf.AfterK

	plainInput := ciphertextToFloat(input, cc)

	outputChannel := cf.KernelNumber
	outputHeight := cf.InputDataHeight / cf.Stride
	outputWidth := cf.InputDataWidth / cf.Stride

	// Input 3d data is packed version of unpacked-input 3d data (=output3d)
	inputChannel := outputChannel / (k * k)
	inputHeight := outputHeight * k
	inputWidth := outputWidth * k

	// for i := 0; i < len(plainInput); i++ {
	// 	if plainInput[i] > 0.00001 || plainInput[i] < -0.00001 {
	// 		fmt.Printf("%.2f ", plainInput[i])
	// 	} else {
	// 		fmt.Print("0 ")
	// 	}
	// }

	// Make it UnParallel
	input3d := make([][][]float64, inputChannel)
	cur := 0
	for c := 0; c < inputChannel; c++ {
		input3d[c] = make([][]float64, inputHeight)
		for h := 0; h < inputHeight; h++ {
			input3d[c][h] = make([]float64, inputWidth)
			for w := 0; w < inputWidth; w++ {
				input3d[c][h][w] = plainInput[cur]
				cur += 1
			}
		}
	}

	// Initialize output3d
	output3d := make([][][]float64, outputChannel)
	for c := 0; c < outputChannel; c++ {
		output3d[c] = make([][]float64, outputHeight)
		for h := 0; h < outputHeight; h++ {
			output3d[c][h] = make([]float64, outputWidth)
		}
	}

	// Make plain output3d
	for c := 0; c < outputChannel; c++ {
		for h := 0; h < outputHeight; h++ {
			for w := 0; w < outputWidth; w++ {
				output3d[c][h][w] = input3d[c/(k*k)][h*k+(c%(k*k))/k][w*k+(c%(k*k))%k]
			}
		}
	}

	return output3d
}
func mse(a, b []float64) float64 {
	var sum float64
	for i := range a {
		diff := a[i] - b[i]
		sum += diff * diff
	}
	return sum / float64(len(a))
}

func relativeError(a, b []float64) float64 {
	var sum float64
	for i := range a {
		if a[i] != 0 {
			sum += math.Abs((a[i] - b[i]) / a[i])
		}
	}
	return sum / float64(len(a))
}

func bitAccuracy(a, b []float64) float64 {

	var sameBits int
	for i := range a {
		aBits := math.Float64bits(a[i])
		bBits := math.Float64bits(b[i])
		diffBits := aBits ^ bBits
		sameBits += 64 - bits.OnesCount64(diffBits)
	}
	return float64(sameBits) / float64(len(a))
}

func InfinityNormDiff(a, b []float64) float64 {
	if len(a) != len(b) {
		panic("Vectors must be of the same length")
	}
	maxDiff := 0.0
	for i := range a {
		diff := math.Abs(a[i] - b[i])
		if diff > maxDiff {
			maxDiff = diff
		}
	}
	return maxDiff
}

// function to calculate accuracy, recall, f1-score
func MSE_RE_infNorm(trueVal [][][]float64, predictVal [][][]float64) []float64 {
	trueFlat := flatten(trueVal)
	predFlat := flatten(predictVal)

	if len(trueFlat) != len(predFlat) {
		fmt.Printf("true val len :%v, FHE val len :%v\n", len(trueFlat), len(predFlat))
		panic("MSE_RE_bitAcc : Length mismatch between true values and predicted values")
	}

	//for debug
	for i := 0; i < len(trueFlat); i++ {
		if (trueFlat[i]-predFlat[i])*(trueFlat[i]-predFlat[i]) > 0.1 {
			fmt.Printf("%v ", i)
		}
	}

	mseVal := mse(trueFlat, predFlat)
	reVal := relativeError(trueFlat, predFlat)
	infNormVal := InfinityNormDiff(trueFlat, predFlat)

	return []float64{mseVal, reVal, infNormVal}

}

func copyPaste(input []float64, copyNum int) []float64 {
	var result []float64
	for i := 0; i < copyNum; i++ {
		for _, value := range input {
			result = append(result, value)
		}
	}
	return result
}
func ErrorPrint(err error) {
	if err != nil {
		fmt.Println(err)
	}
}
func kernelTxtToVector(inputFilePath string) []float64 {
	file, err := os.Open(inputFilePath)
	if err != nil {
		fmt.Println("Error:", err)
		return nil
	}
	defer file.Close()

	var floats []float64

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		floatVal, err := strconv.ParseFloat(scanner.Text(), 64)
		if err != nil {
			fmt.Println("Error:", err)
			return nil
		}

		floats = append(floats, floatVal)
	}

	return floats
}

func FloatToTxt(filePath string, floats []float64) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		file, err := os.Create(filePath)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer file.Close()

		for _, val := range floats {
			_, err := file.WriteString(fmt.Sprintf("%.15f\n", val))
			if err != nil {
				fmt.Println(err)
				return
			}
		}

		fmt.Printf("File '%s' created successfully.\n", filePath)
	} else {
		file, err := os.Create(filePath)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer file.Close()

		for _, val := range floats {
			_, err := file.WriteString(fmt.Sprintf("%.15f\n", val))
			if err != nil {
				fmt.Println(err)
				return
			}
		}
		fmt.Printf("File '%s' already exists. Overwrited\n", filePath)
	}

}

func txtToFloat(txtPath string) []float64 {
	file, err := os.Open(txtPath)
	if err != nil {
		return nil
	}
	defer file.Close()

	var floats []float64

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		floatVal, err := strconv.ParseFloat(scanner.Text(), 64)
		if err != nil {
			return nil
		}

		floats = append(floats, floatVal)
	}

	if err := scanner.Err(); err != nil {
		return nil
	}

	// // convert to complex
	// complexInput := convertFloatToComplex(floats)

	// // encode to Plaintext
	// exPlain := encoder.EncodeNew(complexInput, params.MaxLevel(), params.DefaultScale(), params.LogSlots())

	return floats
}

func plainRelu(inputFloat []float64) []float64 {
	var outputFloat []float64
	for i := 0; i < len(inputFloat); i++ {
		if inputFloat[i] < 0 {
			outputFloat = append(outputFloat, 0)
		} else {
			outputFloat = append(outputFloat, inputFloat[i])
		}
	}
	return outputFloat
}

func append0(inputFloat []float64, aimLen int) []float64 {
	var outputFloat []float64
	for i := 0; i < aimLen; i++ {
		if i < len(inputFloat) {
			outputFloat = append(outputFloat, inputFloat[i])
		} else {
			outputFloat = append(outputFloat, 0)
		}
	}
	return outputFloat
}

func GetFirstLocate(channel int, sameCopy int, k int) int {
	ctLen := 32768
	copyNum := 2
	if k == 4 {
		copyNum = 8
	} else if k == 2 {
		copyNum = 4
	}

	locate := channel%k + channel%(k*k)/k*32 + channel/(k*k)*1024 + (ctLen/copyNum)*sameCopy

	return locate
}
func getPrettyMatrix(h int, w int) [][]float64 {
	result := make([][]float64, h)
	for i := range result {
		result[i] = make([]float64, w)
	}

	for i := 0; i < h; i++ {
		for j := 0; j < w; j++ {
			result[i][j] = float64(10*i+j) / 100.0
		}
	}
	return result
}
func originalMatMul(A, B [][]float64) [][]float64 {
	rowsA := len(A)
	colsA := len(A[0])
	colsB := len(B[0])

	C := make([][]float64, rowsA)
	for i := range C {
		C[i] = make([]float64, colsB)
	}

	for i := 0; i < rowsA; i++ {
		for j := 0; j < colsB; j++ {
			sum := 0.0
			for k := 0; k < colsA; k++ {
				sum += A[i][k] * B[k][j]
			}
			C[i][j] = sum
		}
	}
	return C
}
func make2dTo1d(B [][]float64) []float64 {
	result := make([]float64, len(B))
	for i := 0; i < len(B); i++ {
		result[i] = B[i][0]
	}
	return result
}
func resize(A []float64, ctLength int) []float64 {
	diff := ctLength - len(A)
	if diff <= 0 {
		return A
	}
	for i := 0; i < diff; i++ {
		A = append(A, 0)
	}
	return A
}
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func TimeDurToFloatSec(inputTime time.Duration) float64 {
	return float64(inputTime.Nanoseconds()) / 1e9
}

func TimeDurToFloatMiliSec(inputTime time.Duration) float64 {
	return float64(inputTime.Nanoseconds()) / 1e6
}

func TimeDurToFloatNanoSec(inputTime time.Duration) float64 {
	return float64(inputTime.Nanoseconds())
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
func ciphertextToFloat(exCipher *rlwe.Ciphertext, cc *customContext) []float64 {

	// Decrypt to Plaintext
	exPlain := cc.Decryptor.DecryptNew(exCipher)

	// Decode to []complex128
	float := make([]float64, cc.Params.MaxSlots())
	cc.Encoder.Decode(exPlain, float)

	return float
}

// helper function
func minMaxAvg(values []float64) (min, max, avg float64) {
	if len(values) == 0 {
		return 0, 0, 0
	}
	min, max, sum := values[0], values[0], 0.0
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
		sum += v
	}
	avg = sum / float64(len(values))
	return
}
