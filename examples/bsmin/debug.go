package main

import (
	"bufio"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"os"
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
			if currentZero { //0이였었는데 1이 나옴
				currentZero = false
				fmt.Printf("0 : %v\n", count)
				count = 1
			} else { //1이였었는데 0이 나옴
				currentZero = true
				fmt.Printf("1 : %v\n", count)
				count = 1
			}
		}
	}
	if currentZero { //0이였었는데 1이 나옴
		currentZero = false
		fmt.Printf("0 : %v\n", count)
		count = 0
	} else { //1이였었는데 0이 나옴
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
	// 파일이 이미 존재하는지 확인
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// 파일이 존재하지 않으면 생성
		file, err := os.Create(filePath)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer file.Close()

		// float 배열의 각 값 저장
		for _, val := range floats {
			// float 값을 문자열로 변환하여 파일에 쓰기
			_, err := file.WriteString(fmt.Sprintf("%.15f\n", val))
			if err != nil {
				fmt.Println(err)
				return
			}
		}

		fmt.Printf("File '%s' created successfully.\n", filePath)
	} else {
		// 파일이 존재하지 않으면 생성
		file, err := os.Create(filePath)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer file.Close()

		// float 배열의 각 값 저장
		for _, val := range floats {
			// float 값을 문자열로 변환하여 파일에 쓰기
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
func makeRandomData(width, height, channel int) [][][]float64 {
	// 벡터 생성
	randomVector := make([][][]float64, channel)
	for d := 0; d < channel; d++ {
		randomVector[d] = make([][]float64, height)
		for i := 0; i < height; i++ {
			randomVector[d][i] = make([]float64, width)
			for j := 0; j < width; j++ {
				randomVector[d][i][j] = float64(rand.Intn(256))
			}
		}
	}

	return randomVector
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
func Convolution3D(inputImage [][][]float64, filter [][][]float64, stride int) [][]float64 {
	// 입력 이미지와 필터의 크기 계산
	imageDepth, imageRows, imageCols := len(inputImage), len(inputImage[0]), len(inputImage[0][0])
	filterDepth, filterRows, filterCols := len(filter), len(filter[0]), len(filter[0][0])

	// 결과 이미지 크기 계산
	resultRows := (imageRows-filterRows)/stride + 1
	resultCols := (imageCols-filterCols)/stride + 1

	// 결과 이미지 초기화
	result := make([][]float64, resultRows)
	for i := range result {
		result[i] = make([]float64, resultCols)
	}

	// 3D 컨볼루션 연산 수행
	for i := 0; i < resultRows; i++ {
		for j := 0; j < resultCols; j++ {
			// 각 채널별로 컨볼루션 연산 수행
			for d := 0; d < imageDepth; d++ {
				for k := 0; k < filterDepth; k++ {
					for l := 0; l < filterRows; l++ {
						for m := 0; m < filterCols; m++ {
							result[i][j] += inputImage[d][i*stride+l][j*stride+m] * filter[k][l][m]
						}
					}
				}
			}
		}
	}

	return result
}

func Convolution3DMulKernels(inputImage [][][]float64, kernels [][][][]float64, stride int) [][][]float64 {
	// 입력 이미지와 커널의 크기 계산
	imageDepth, imageRows, imageCols := len(inputImage), len(inputImage[0]), len(inputImage[0][0])
	kernelDepth, kernelRows, kernelCols, numKernels := len(kernels[0]), len(kernels[0][0]), len(kernels[0][0][0]), len(kernels)

	// 결과 이미지 크기 계산
	resultRows := (imageRows-kernelRows)/stride + 1
	resultCols := (imageCols-kernelCols)/stride + 1

	// 결과 이미지 초기화
	result := make([][][]float64, numKernels)
	for i := range result {
		result[i] = make([][]float64, resultRows)
		for j := range result[i] {
			result[i][j] = make([]float64, resultCols)
		}
	}

	// 3D 컨볼루션 연산 수행
	for k := 0; k < numKernels; k++ {
		for i := 0; i < resultRows; i++ {
			for j := 0; j < resultCols; j++ {
				// 각 채널별로 컨볼루션 연산 수행
				for d := 0; d < imageDepth; d++ {
					for l := 0; l < kernelDepth; l++ {
						for m := 0; m < kernelRows; m++ {
							for n := 0; n < kernelCols; n++ {
								result[k][i][j] += inputImage[d][i*stride+m][j*stride+n] * kernels[k][l][m][n]
							}
						}
					}
				}
			}
		}
	}

	return result
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

// 2차원 float 배열 출력 함수
func print2DArray(arr [][]float64) {
	for i := 0; i < len(arr); i++ {
		for j := 0; j < len(arr[i]); j++ {
			fmt.Printf("%v ", arr[i][j])
		}
		fmt.Println()
	}
}

// 3차원 float 배열 출력 함수
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
	for zz := 0; zz < len(input); zz++ {
		for yy := 0; yy < len(input[0]); yy++ {
			for xx := 0; xx < len(input[0][0]); xx++ {
				result = append(result, input[zz][yy][xx])
			}
		}
	}
	return result
}
func packingWithWidth(input []float64, dataWidth int, k int) []float64 {
	if k == 1 {
		return input
	}

	beforeChannel := len(input) / (dataWidth * dataWidth)
	input3d := make([][][]float64, beforeChannel)
	output3d := make([][][]float64, beforeChannel/(k*k))

	for z := 0; z < beforeChannel; z++ {
		input3d[z] = make([][]float64, dataWidth)
		for y := 0; y < dataWidth; y++ {
			input3d[z][y] = make([]float64, dataWidth)
			for x := 0; x < dataWidth; x++ {
				input3d[z][y][x] = input[z*dataWidth*dataWidth+y*dataWidth+x]
			}
		}
	}

	for z := 0; z < beforeChannel/(k*k); z++ {
		output3d[z] = make([][]float64, dataWidth*k)
		for y := 0; y < dataWidth*k; y++ {
			output3d[z][y] = make([]float64, dataWidth*k)
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
func packing(input []float64, k int) []float64 {
	if k == 1 {
		return input
	}

	dataWidth := 32

	if k == 2 {
		dataWidth = 16
	} else if k == 4 {
		dataWidth = 8
	}

	beforeChannel := len(input) / (dataWidth * dataWidth)
	input3d := make([][][]float64, beforeChannel)
	output3d := make([][][]float64, beforeChannel/(k*k))

	for z := 0; z < beforeChannel; z++ {
		input3d[z] = make([][]float64, dataWidth)
		for y := 0; y < dataWidth; y++ {
			input3d[z][y] = make([]float64, dataWidth)
			for x := 0; x < dataWidth; x++ {
				input3d[z][y][x] = input[z*dataWidth*dataWidth+y*dataWidth+x]
			}
		}
	}

	for z := 0; z < beforeChannel/(k*k); z++ {
		output3d[z] = make([][]float64, dataWidth*k)
		for y := 0; y < dataWidth*k; y++ {
			output3d[z][y] = make([]float64, dataWidth*k)
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

func unpacking(input []float64, dataWidth, k int) []float64 {
	if k == 1 {
		return input
	}
	afterChannel := len(input) / (dataWidth * dataWidth)
	output3d := make([][][]float64, afterChannel)
	for z := 0; z < afterChannel; z++ {
		output3d[z] = make([][]float64, dataWidth)
		for yy := 0; yy < dataWidth; yy++ {
			output3d[z][yy] = make([]float64, dataWidth)
			for xx := 0; xx < dataWidth; xx++ {
				output3d[z][yy][xx] = input[z/k/k*32*32+yy*k*32+(z%(k*k))/k*32+xx*k+(z%(k*k))%k]
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
	// 파일 열기
	file, err := os.Open(inputFilePath)
	if err != nil {
		fmt.Println("Error:", err)
		return nil
	}
	defer file.Close()

	var floats []float64

	// 파일 스캐너 생성
	scanner := bufio.NewScanner(file)

	// 각 줄 읽어오기
	for scanner.Scan() {
		// 문자열을 float64로 변환
		floatVal, err := strconv.ParseFloat(scanner.Text(), 64)
		if err != nil {
			fmt.Println("Error:", err)
			return nil
		}

		// 슬라이스에 추가
		floats = append(floats, floatVal)
	}

	return floats
}

func FloatToTxt(filePath string, floats []float64) {
	// 파일이 이미 존재하는지 확인
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// 파일이 존재하지 않으면 생성
		file, err := os.Create(filePath)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer file.Close()

		// float 배열의 각 값 저장
		for _, val := range floats {
			// float 값을 문자열로 변환하여 파일에 쓰기
			_, err := file.WriteString(fmt.Sprintf("%.15f\n", val))
			if err != nil {
				fmt.Println(err)
				return
			}
		}

		fmt.Printf("File '%s' created successfully.\n", filePath)
	} else {
		// 파일이 존재하지 않으면 생성
		file, err := os.Create(filePath)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer file.Close()

		// float 배열의 각 값 저장
		for _, val := range floats {
			// float 값을 문자열로 변환하여 파일에 쓰기
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
	// 파일 열기
	file, err := os.Open(txtPath)
	if err != nil {
		return nil
	}
	defer file.Close()

	var floats []float64

	// 파일 스캐너 생성
	scanner := bufio.NewScanner(file)

	// 각 줄 읽어오기
	for scanner.Scan() {
		// 문자열을 float64로 변환
		floatVal, err := strconv.ParseFloat(scanner.Text(), 64)
		if err != nil {
			return nil
		}

		// 슬라이스에 추가
		floats = append(floats, floatVal)
	}

	// 스캔 중 에러 확인
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
