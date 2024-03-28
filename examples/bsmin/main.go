package main

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"rotOptResnet/mulParModules"
	"sort"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/tuneinsight/lattigo/v5/core/rlwe"
	"github.com/tuneinsight/lattigo/v5/schemes/ckks"
)

type CIFAR10Image struct {
	Label byte
	Data  [3072]byte // 32x32 이미지 (3채널, 총 3072 바이트)
}

type customContext struct {
	Params      ckks.Parameters
	Encoder     *ckks.Encoder
	Kgen        *rlwe.KeyGenerator
	Sk          *rlwe.SecretKey
	Pk          *rlwe.PublicKey
	EncryptorPk *rlwe.Encryptor
	EncryptorSk *rlwe.Encryptor
	Decryptor   *rlwe.Decryptor
	Evaluator   *ckks.Evaluator
}

// ///////////////////////////////
// func txtToPlain(txtPath string) (*rlwe.Plaintext, error) {
// 	// 파일 열기
// 	file, err := os.Open(txtPath)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer file.Close()

// 	var floats []float64

// 	// 파일 스캐너 생성
// 	scanner := bufio.NewScanner(file)

// 	// 각 줄 읽어오기
// 	for scanner.Scan() {
// 		// 문자열을 float64로 변환
// 		floatVal, err := strconv.ParseFloat(scanner.Text(), 64)
// 		if err != nil {
// 			return nil, err
// 		}

// 		// 슬라이스에 추가
// 		floats = append(floats, floatVal)
// 	}

// 	// 스캔 중 에러 확인
// 	if err := scanner.Err(); err != nil {
// 		return nil, err
// 	}

// 	// convert to complex
// 	complexInput := convertFloatToComplex(floats)

// 	// encode to Plaintext
// 	exPlain := encoder.EncodeNew(complexInput, params.MaxLevel(), params.DefaultScale(), params.LogSlots())

// 	return exPlain, nil
// }

// func combinePlainMap(originalMap map[string]*rlwe.Plaintext, newList []string) {
// 	for _, path := range newList {
// 		plain, err := txtToPlain(path)
// 		if err != nil {
// 			fmt.Println("Error:", err)
// 			return
// 		}

//			originalMap[path] = plain
//		}
//	}
func floatToCiphertext(floatInput []float64, params ckks.Parameters, encoder *ckks.Encoder, encryptor *rlwe.Encryptor) *rlwe.Ciphertext {

	// encode to Plaintext
	exPlain := ckks.NewPlaintext(params, params.MaxLevel())
	encoder.Encode(floatInput, exPlain)

	// Encrypt to Ciphertext
	exCipher, err := encryptor.EncryptNew(exPlain)
	ErrorPrint(err)

	return exCipher
}
func floatToCiphertextLevel(floatInput []float64, level int, params ckks.Parameters, encoder *ckks.Encoder, encryptor *rlwe.Encryptor) *rlwe.Ciphertext {

	// encode to Plaintext
	exPlain := ckks.NewPlaintext(params, level)
	_ = encoder.Encode(floatInput, exPlain)

	// Encrypt to Ciphertext
	exCipher, err := encryptor.EncryptNew(exPlain)
	if err != nil {
		fmt.Println(err)
	}

	return exCipher
}

func ciphertextToFloat(exCipher *rlwe.Ciphertext, cc *customContext) []float64 {

	// Decrypt to Plaintext
	exPlain := cc.Decryptor.DecryptNew(exCipher)

	// Decode to []complex128
	float := make([]float64, cc.Params.MaxSlots())
	cc.Encoder.Decode(exPlain, float)

	return float
}
func printCipherSample(message string, cipherInput *rlwe.Ciphertext, cc *customContext, start int, end int) {
	plain := cc.Decryptor.DecryptNew(cipherInput)
	floatOutput := make([]float64, cc.Params.MaxSlots())
	cc.Encoder.Decode(plain, floatOutput)

	msgSample1DArray(message, floatOutput, 0, 10)
}

// func basicTest() {

// 	if err != nil {
// 		panic(err)
// 	}

// 	////////////Basic Operation//////////////
// 	// Generate a random float input
// 	floatInput := makeRandomFloat(params.Slots())

// 	// print input
// 	fmt.Println("Input : ")
// 	sample1DArray(floatInput, 0, 10)

// 	// convert to complex
// 	complexInput := convertFloatToComplex(floatInput)

// 	// encode to Plaintext
// 	exPlain := encoder.EncodeNew(complexInput, params.MaxLevel(), params.DefaultScale(), params.LogSlots())

// 	// Encrypt to Ciphertext
// 	exCipher := encryptor.EncryptNew(exPlain)

// 	// Rotate Ciphertext
// 	printCipherSample("exCipher : ", exCipher, 0, 10)
// 	exCipher2 := evaluator.RotateNew(exCipher, -2)
// 	printCipherSample("exCipher2(rot -2) : ", exCipher2, 0, 10)
// 	exCipher2 = evaluator.AddNew(exPlain, exCipher2)
// 	printCipherSample("exCipher2(sum) : ", exCipher2, 0, 10)

// 	// Decrypt to Plaintext
// 	exPlain2 := decryptor.DecryptNew(exCipher)

// 	// Decode to []complex128
// 	result := encoder.Decode(exPlain2, params.LogSlots())

// 	// convert to float
// 	floatOutput := convertComplexToFloat(result)

// 	//print float output
// 	fmt.Println("Output : ")
// 	sample1DArray(floatOutput, 0, 10)
// 	//sample1DComplex(result, 0, 10)

// 	// eucldiean distance
// 	ed := euclideanDistance(floatOutput, floatInput)
// 	fmt.Println("Euclidean Distance : ", ed)

// }
func avgPoolTest(cc *customContext) {
	//register
	rot := mulParModules.AvgPoolRegister()

	//rot register
	newEvaluator := RotIndexToGaloisElements(rot, cc)

	//make avgPooling instance
	avgPooling := mulParModules.NewAvgPool(newEvaluator, cc.Encoder, cc.Params)

	//Make input float data
	inputFloat := makeRandomFloat(cc.Params.MaxSlots())

	//Encryption
	inputCt := floatToCiphertext(inputFloat, cc.Params, cc.Encoder, cc.EncryptorSk)

	//Timer start
	startTime := time.Now()

	//AvgPooling Foward
	outputCt := avgPooling.Foward(inputCt)

	//Timer end
	endTime := time.Now()

	//Print Elapsed Time
	fmt.Printf("Time : %v \n", endTime.Sub(startTime))

	//Decryption
	outputFloat := ciphertextToFloat(outputCt, cc)

	// sample1DArray(outputFloat, 0, 32768)
	sample1DArray(outputFloat, 0, 100)

}

func fullyConnectedTest(layerNum int, cc *customContext) {
	//register
	rot := mulParModules.ParFCRegister()

	//rot register
	newEvaluator := RotIndexToGaloisElements(rot, cc)

	//make avgPooling instance
	fc := mulParModules.NewparFC(newEvaluator, cc.Encoder, cc.Params, layerNum)

	//Make input float data
	inputFloat := makeRandomFloat(cc.Params.MaxSlots())

	//Encryption
	inputCt := floatToCiphertext(inputFloat, cc.Params, cc.Encoder, cc.EncryptorSk)

	//Timer start
	startTime := time.Now()

	//AvgPooling Foward
	outputCt := fc.Foward(inputCt)

	//Timer end
	endTime := time.Now()

	//Print Elapsed Time
	fmt.Printf("Time : %v \n", endTime.Sub(startTime))

	//Decryption
	outputFloat := ciphertextToFloat(outputCt, cc)

	// sample1DArray(outputFloat, 0, 32768)
	sample1DArray(outputFloat, 0, 10)
}

func rotOptDownSamplingTest(cc *customContext) {
	//register
	rot := mulParModules.RotOptDSRegister()

	//rot register
	newEvaluator := RotIndexToGaloisElements(rot, cc)

	//make avgPooling instance
	ds16 := mulParModules.NewRotOptDS(16, newEvaluator, cc.Encoder, cc.Params)
	ds32 := mulParModules.NewRotOptDS(32, newEvaluator, cc.Encoder, cc.Params)

	//Make input float data
	inputFloat := makeRandomFloat(cc.Params.MaxSlots())

	for level := 2; level <= cc.Params.MaxLevel(); level++ {
		// Encryption
		inputCt := floatToCiphertextLevel(inputFloat, level, cc.Params, cc.Encoder, cc.EncryptorSk)
		// /////////
		// Timer start
		startTime := time.Now()

		// AvgPooling Foward
		ds16.Foward(inputCt)

		// Timer end
		endTime := time.Now()

		// Print Elapsed Time
		fmt.Printf("Time(16) : %v \n", endTime.Sub(startTime))
		// ////////
		// Timer start
		startTime = time.Now()

		// AvgPooling Foward
		ds32.Foward(inputCt)

		// Timer end
		endTime = time.Now()

		// Print Elapsed Time
		fmt.Printf("Time(32) : %v \n", endTime.Sub(startTime))
	}

	// 	//Decryption
	// 	outputFloat16 := ciphertextToFloat(outputCt16, cc)
	// 	outputFloat32 := ciphertextToFloat(outputCt32, cc)

	// //Test
	// fmt.Println("==16==")
	// count01num(outputFloat16)
	// fmt.Println("\n==32==")
	// count01num(outputFloat32)
}
func mulParDownSamplingTest(cc *customContext) {
	//register
	rot := mulParModules.MulParDSRegister()

	//rot register
	newEvaluator := RotIndexToGaloisElements(rot, cc)

	//make avgPooling instance
	ds16 := mulParModules.NewMulParDS(16, newEvaluator, cc.Encoder, cc.Params)
	ds32 := mulParModules.NewMulParDS(32, newEvaluator, cc.Encoder, cc.Params)

	//Make input float data
	inputFloat := makeRandomFloat(cc.Params.MaxSlots())

	for level := 1; level <= cc.Params.MaxLevel(); level++ {
		fmt.Println("===", level, "===")
		// Encryption
		inputCt := floatToCiphertextLevel(inputFloat, level, cc.Params, cc.Encoder, cc.EncryptorSk)
		// /////////
		// Timer start
		startTime := time.Now()

		// AvgPooling Foward
		ds16.Foward(inputCt)

		// Timer end
		endTime := time.Now()

		// Print Elapsed Time
		fmt.Printf("Time(16) : %v \n", endTime.Sub(startTime))
		// ////////
		// Timer start
		startTime = time.Now()

		// AvgPooling Foward
		ds32.Foward(inputCt)

		// Timer end
		endTime = time.Now()

		// Print Elapsed Time
		fmt.Printf("Time(32) : %v \n", endTime.Sub(startTime))
	}

	// 	//Decryption
	// 	outputFloat16 := ciphertextToFloat(outputCt16, cc)
	// 	outputFloat32 := ciphertextToFloat(outputCt32, cc)

	// //Test
	// fmt.Println("==16==")
	// count01num(outputFloat16)
	// fmt.Println("\n==32==")
	// count01num(outputFloat32)
}

// func mulParConvTest(layerNum int, cc *customContext, startCipherLevel int) {
// 	// mulParModules.MakeTxtRotOptConvWeight()
// 	// mulParModules.MakeTxtRotOptConvFilter()
// 	// convIDs := []string{"CONV1", "CONV2", "CONV3s2", "CONV3", "CONV4s2", "CONV4"}
// 	// maxDepth := []int{2, 2, 2, 2, 2, 2}
// 	convIDs := []string{"CONV4s2"}
// 	maxDepth := []int{2}
// 	iter := 1

// 	for index := 0; index < len(convIDs); index++ {
// 		for depth := 2; depth < maxDepth[index]+1; depth++ {
// 			convID := convIDs[index]
// 			fmt.Printf("convID : %s, Depth : %v, iter : %v\n", convID, depth, iter)

// 			/////Real Convolution/////
// 			cf := mulParModules.GetConvFeature(convID)
// 			inputRandomVector := makeRandomData(cf.InputDataWidth, cf.InputDataHeight, cf.InputDataChannel)

// 			flattenInputData := flatten(inputRandomVector)

// 			if cf.InputDataChannel == 3 {
// 				for i := 0; i < 1024; i++ {
// 					flattenInputData = append(flattenInputData, 0)
// 				}
// 			}

// 			flattenInputData = copyPaste(flattenInputData, cf.BeforeCopy)

// 			if len(flattenInputData) != 32768 {
// 				fmt.Println("You set the wrong parameter!")
// 			}

// 			flattenInputData = packingWithWidth(flattenInputData, cf.InputDataWidth, cf.K)

// 			kernel_filePath := "mulParModules/precomputed/resnetPtParam/" + strconv.Itoa(layerNum) + "/" + getConvTestTxtPath(convID)
// 			fmt.Println(kernelTxtToVector(kernel_filePath))
// 			flattenOriginal := flatten(realConv(cf.InputDataWidth, cf.InputDataHeight, cf.InputDataChannel, cf.KernelSize, cf.KernelNumber, cf.Stride, inputRandomVector, kernelTxtToVector(kernel_filePath)))
// 			flattenOriginal = copyPaste(flattenOriginal, cf.AfterCopy)
// 			flattenOriginal = packingWithWidth(flattenOriginal, cf.InputDataWidth/cf.Stride, cf.K*cf.Stride)
// 			if len(flattenOriginal) != 32768 {
// 				fmt.Println("You have set wrong parameter!")
// 			}

// 			////CKKS convolution////
// 			d := startCipherLevel
// 			if depth > startCipherLevel {
// 				d = depth
// 			}
// 			plain := ckks.NewPlaintext(cc.Params, d)
// 			cc.Encoder.Encode(flattenInputData, plain)
// 			// inputCt, _ := cc.EncryptorSk.EncryptNew(plain)

// 			//register
// 			rot := mulParModules.MulParConvRegister(convID)
// 			// for _, r := range rot {
// 			// 	fmt.Println(len(r), r)
// 			// }

// 			// for i := 0; i < 3; i++ {
// 			// 	fmt.Println(len(rot[i]))
// 			// }

// 			//rot register
// 			newEvaluator := rotIndexToGaloisEl(int2dTo1d(rot), cc.Params, cc.Kgen, cc.Sk)

// 			//make rotOptConv instance
// 			conv := mulParModules.NewMulParConv(newEvaluator, cc.Encoder, cc.Decryptor, cc.Params, layerNum, convID, depth, getConvTestNum(convID)[0], getConvTestNum(convID)[1])

// 			var outputCt *rlwe.Ciphertext
// 			//Timer start
// 			startTime := time.Now()

// 			//Conv Foward
// 			// for i := 0; i < iter; i++ {
// 			// 	outputCt = conv.Foward(inputCt)
// 			// }
// 			for level := 2; level <= cc.Params.MaxLevel(); level++ {
// 				plain := ckks.NewPlaintext(cc.Params, level)
// 				cc.Encoder.Encode(flattenInputData, plain)
// 				inputCt, _ := cc.EncryptorSk.EncryptNew(plain)
// 				startTime := time.Now()
// 				outputCt = conv.Foward(inputCt)
// 				endTime := time.Now()

// 				fmt.Printf("( %d, %v) \n", level, endTime.Sub(startTime)/time.Duration(iter))
// 			}

// 			//Timer end
// 			endTime := time.Now()

// 			//Print Elapsed Time
// 			fmt.Printf("Time : %v \n", endTime.Sub(startTime)/time.Duration(iter))

// 			//Decryption
// 			outputFloat := ciphertextToFloat(outputCt, cc)

// 			// fmt.Println(euclideanDistance(unpacking(outputFloat, cf.InputDataWidth/cf.Stride, cf.K*cf.Stride), flattenOriginal))
// 			euclideanDistance(outputFloat, flattenOriginal)

// 			floatToTxt("outputFloat", outputFloat)
// 			floatToTxt("flattenOriginal", flattenOriginal)

//				// // sample1DArray(outputFloat, 0, 32768)
//				// sample1DArray(outputFloat, 0, 32768)
//			}
//		}
//	}
func MakeGalois(cc *customContext, rotIndexes [][]int) [][]*rlwe.GaloisKey {

	galEls := make([][]*rlwe.GaloisKey, len(rotIndexes))

	for level := 0; level < len(rotIndexes); level++ {
		var galElements []uint64
		for _, rot := range rotIndexes[level] {
			galElements = append(galElements, cc.Params.GaloisElement(rot))
		}
		galKeys := cc.Kgen.GenGaloisKeysNew(galElements, cc.Sk)

		galEls = append(galEls, galKeys)
		//일단 48 byte 인듯
		fmt.Println(unsafe.Sizeof(*galKeys[0]), unsafe.Sizeof(galKeys[0].GaloisElement), unsafe.Sizeof(galKeys[0].NthRoot), unsafe.Sizeof(galKeys[0].EvaluationKey), unsafe.Sizeof(galKeys[0].GadgetCiphertext), unsafe.Sizeof(galKeys[0].BaseTwoDecomposition), unsafe.Sizeof(galKeys[0].Value))
	}
	// newEvaluator := ckks.NewEvaluator(cc.Params, rlwe.NewMemEvaluationKeySet(cc.Kgen.GenRelinearizationKeyNew(cc.Sk), galKeys...))
	return galEls
}
func ClientMakeGaloisWithLevel(cc *customContext, rotIndexes [][]int) [][]*rlwe.GaloisKey {

	galEls := make([][]*rlwe.GaloisKey, len(rotIndexes))

	for level := 0; level < len(rotIndexes); level++ {
		var galElements []uint64
		for _, rot := range rotIndexes[level] {
			galElements = append(galElements, cc.Params.GaloisElement(rot))
		}
		galKeys := cc.Kgen.ClientGenGaloisKeysNew(level, galElements, cc.Sk)

		galEls = append(galEls, galKeys)
		//일단 48 byte 인듯
		fmt.Println(unsafe.Sizeof(*galKeys[0]), unsafe.Sizeof(galKeys[0].GaloisElement), unsafe.Sizeof(galKeys[0].NthRoot), unsafe.Sizeof(galKeys[0].EvaluationKey), unsafe.Sizeof(galKeys[0].GadgetCiphertext), unsafe.Sizeof(galKeys[0].BaseTwoDecomposition), unsafe.Sizeof(galKeys[0].Value))
	}
	// newEvaluator := ckks.NewEvaluator(cc.Params, rlwe.NewMemEvaluationKeySet(cc.Kgen.GenRelinearizationKeyNew(cc.Sk), galKeys...))
	return galEls
}

func rotOptConvTest(layerNum int, cc *customContext, startCipherLevel int) {
	// mulParModules.MakeTxtRotOptConvWeight()
	// mulParModules.MakeTxtRotOptConvFilter()
	// convIDs := []string{"CONV1", "CONV2", "CONV3s2", "CONV3", "CONV4s2", "CONV4"}
	// convIDs := []string{"CONV3", "CONV4s2"}
	convIDs := []string{"CONV2"}
	maxDepth := []int{2}
	// maxDepth := []int{2, 4, 5, 4, 5, 4}
	// maxDepth := []int{2, 2}
	// maxDepth := []int{2}

	iter := 1

	startDepth := 2

	for index := 0; index < len(convIDs); index++ {
		for depth := startDepth; depth < maxDepth[index]+1; depth++ { //원래 depth:=2
			convID := convIDs[index]

			fmt.Printf("=== convID : %s, Depth : %v, iter : %v === \n", convID, depth, iter)

			/////Real Convolution/////
			cf := mulParModules.GetConvFeature(convID)
			inputRandomVector := makeRandomData(cf.InputDataWidth, cf.InputDataHeight, cf.InputDataChannel)

			flattenInputData := flatten(inputRandomVector)

			if cf.InputDataChannel == 3 {
				for i := 0; i < 1024; i++ {
					flattenInputData = append(flattenInputData, 0)
				}
			}

			flattenInputData = copyPaste(flattenInputData, cf.BeforeCopy)

			if len(flattenInputData) != 32768 {
				fmt.Println("You set the wrong parameter!")
			}

			flattenInputData = packingWithWidth(flattenInputData, cf.InputDataWidth, cf.K)

			kernel_filePath := "mulParModules/precomputed/rotOptConv/kernelWeight/" + strconv.Itoa(layerNum) + "/layer" + strconv.Itoa(getConvTestNum(convID)[0]) + "/" + strconv.Itoa(getConvTestNum(convID)[1])
			kernelFloat := kernelTxtToVector(kernel_filePath)
			kernelFloat = unpacking(kernelFloat, cf.InputDataWidth, cf.K)

			kernelSize := 16
			channel := 16
			kernelNum := 3
			index := 0
			// 커널 가중치를 저장할 4차원 배열을 초기화합니다.
			kernelWeight := make([][][][]float64, kernelNum)
			for kn := 0; kn < kernelNum; kn++ {
				kernelWeight[kn] = make([][][]float64, channel)
				for c := 0; c < channel; c++ {
					kernelWeight[kn][c] = make([][]float64, kernelSize)
					for ks1 := 0; ks1 < kernelSize; ks1++ {
						kernelWeight[kn][c][ks1] = make([]float64, kernelSize)
						for ks2 := 0; ks2 < kernelSize; ks2++ {
							kernelWeight[kn][c][ks1][ks2] = kernelFloat[index]
							index++
						}
					}
				}
			}

			flattenOriginal := flatten(realConv(cf.InputDataWidth, cf.InputDataHeight, cf.InputDataChannel, cf.KernelSize, cf.KernelNumber, cf.Stride, inputRandomVector, kernelWeight))
			flattenOriginal = copyPaste(flattenOriginal, cf.AfterCopy)
			flattenOriginal = packingWithWidth(flattenOriginal, cf.InputDataWidth/cf.Stride, cf.K*cf.Stride)
			if len(flattenOriginal) != 32768 {
				fmt.Println("You have set wrong parameter!")
			}

			////CKKS convolution////
			d := startCipherLevel
			if depth > startCipherLevel {
				d = depth
			}
			plain := ckks.NewPlaintext(cc.Params, d)
			cc.Encoder.Encode(flattenInputData, plain)
			inputCt, _ := cc.EncryptorSk.EncryptNew(plain)

			//register
			rots := mulParModules.RotOptConvRegister(convID, depth)
			// for _, r := range rots {
			// 	fmt.Println(len(r), r)
			// }

			//rot register
			newEvaluator := rotIndexToGaloisEl(int2dTo1d(rots), cc.Params, cc.Kgen, cc.Sk)

			//make rotOptConv instance
			conv := mulParModules.NewrotOptConv(newEvaluator, cc.Encoder, cc.Decryptor, cc.Params, layerNum, convID, depth, getConvTestNum(convID)[0], getConvTestNum(convID)[1])

			var outputCt *rlwe.Ciphertext
			//Timer start
			// startTime := time.Now()

			//Conv Foward
			for i := 0; i < iter; i++ {
				outputCt = conv.Foward(inputCt)
			}

			// for level := 2; level <= cc.Params.MaxLevel(); level++ {
			// 	plain := ckks.NewPlaintext(cc.Params, level)
			// 	cc.Encoder.Encode(flattenInputData, plain)
			// 	inputCt, _ := cc.EncryptorSk.EncryptNew(plain)
			// 	startTime := time.Now()
			// 	outputCt = conv.Foward(inputCt)
			// 	endTime := time.Now()

			// 	fmt.Printf("( %d, %v) \n", level, endTime.Sub(startTime)/time.Duration(iter))
			// }

			//Timer end
			// endTime := time.Now()

			//Print Elapsed Time
			// fmt.Printf("Time : %v \n", endTime.Sub(startTime)/time.Duration(iter))
			// fmt.Printf("%v) \n", endTime.Sub(startTime)/time.Duration(iter))

			//Decryption
			outputFloat := ciphertextToFloat(outputCt, cc)

			fmt.Println(euclideanDistance(outputFloat, flattenOriginal))

			floatToTxt("outputFloat", outputFloat)
			floatToTxt("flattenOriginal", flattenOriginal)

			// // sample1DArray(outputFloat, 0, 32768)
			// sample1DArray(outputFloat, 0, 32768)
		}
	}
}
func rotationTimeTest(cc *customContext) {
	iter := 100
	fmt.Println("Time test for rotation iter", iter)
	for level := 0; level <= cc.Params.MaxLevel(); level++ {
		var timeSum int64
		for i := 0; i < iter; i++ {
			// Make input float data
			inputFloat := makeRandomFloat(cc.Params.MaxSlots())

			// Encryption
			inputCt := floatToCiphertextLevel(inputFloat, level, cc.Params, cc.Encoder, cc.EncryptorSk)

			// Timer start
			startTime := time.Now()

			cc.Evaluator.Rotate(inputCt, 2, inputCt)

			// Timer end
			endTime := time.Now()

			timeSum += endTime.Sub(startTime).Milliseconds()
		}
		fmt.Println("level", level, " ", int(timeSum)/iter, "ms")
	}

}
func addMultTimeTest(cc *customContext) {
	iter := 100

	// var err error
	fmt.Println("Time test for iter", iter)

	// Make input float data
	inputFloat := makeRandomFloat(cc.Params.MaxSlots())

	// inputCt3 := floatToCiphertextLevel(inputFloat, cc.Params.MaxLevel()-2, cc.Params, cc.Encoder, cc.EncryptorSk)
	inputCt4 := floatToCiphertextLevel(inputFloat, cc.Params.MaxLevel()-2, cc.Params, cc.Encoder, cc.EncryptorSk)
	for level := cc.Params.MaxLevel() - 2; level > 0; level-- {
		var timeSum float64
		inputCt := floatToCiphertextLevel(inputFloat, level, cc.Params, cc.Encoder, cc.EncryptorSk)
		inputPlain := ckks.NewPlaintext(cc.Params, level)
		cc.Encoder.Encode(inputFloat, inputPlain)
		// inputCt2 := floatToCiphertextLevel(inputFloat, level, cc.Params, cc.Encoder, cc.EncryptorSk)
		for i := 0; i < iter; i++ {
			// fmt.Println("==============")
			// Encryption

			// fmt.Println(ciphertextToFloat(inputCt, cc)[0:10])
			// fmt.Println(ciphertextToFloat(inputCt2, cc)[0:10])

			// Timer start
			startTime := time.Now()

			//Add Case
			// cc.Evaluator.Add(inputCt, inputCt2, inputCt3)

			//Mul Case
			// inputCt3, err := cc.Evaluator.MulNew(inputCt, inputCt2)
			// if err != nil {
			// 	fmt.Println(err)
			// }
			// err = cc.Evaluator.Rescale(inputCt3, inputCt4)

			// if err != nil {
			// 	fmt.Println(err)
			// }

			//Mul with plain
			inputCt3, err := cc.Evaluator.MulNew(inputCt, inputPlain)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println(ciphertextToFloat(inputCt, cc)[0:10])
			values := make([]float64, cc.Params.MaxSlots())
			cc.Encoder.Decode(inputPlain, values)

			fmt.Println(values[0:10])
			fmt.Println(ciphertextToFloat(inputCt3, cc)[0:10])

			cc.Evaluator.Rotate(inputCt3, 2, inputCt3)

			err = cc.Evaluator.Rescale(inputCt3, inputCt4)
			fmt.Println(ciphertextToFloat(inputCt4, cc)[0:10])

			if err != nil {
				fmt.Println(err)
			}

			// Timer end
			endTime := time.Now()

			// fmt.Println(ciphertextToFloat(inputCt4, cc)[0:10])
			// fmt.Println(inputCt4.Level(), inputCt4.Scale)
			timeSum += float64(endTime.Sub(startTime).Milliseconds())
		}
		fmt.Println("level", level, " ", timeSum/float64(iter), "ms")
	}
}

func conv1Test(cc *customContext) {
	//get Float
	exInput := txtToFloat("true_logs/layer0_input_data.txt")

	//Make it to ciphertext
	var copyInput []float64
	for i := 0; i < 8; i++ {
		for j := 0; j < 4096; j++ {
			if j < len(exInput) {
				copyInput = append(copyInput, exInput[j])
			} else {
				copyInput = append(copyInput, 0)
			}
		}
	}

	cipherInput := floatToCiphertextLevel(copyInput, 6, cc.Params, cc.Encoder, cc.EncryptorSk)

	//register
	// rots := mulParModules.RotOptConvRegister("CONV1", 2)
	rots := mulParModules.MulParConvRegister("CONV1")

	fmt.Println(rots)
	// Make new Evaluator with rot indices
	newEvaluator := RotIndexToGaloisElements(int2dTo1d(rots), cc)

	conv := mulParModules.NewMulParConv(newEvaluator, cc.Encoder, cc.Decryptor, cc.Params, 20, "CONV1", 0, 1)

	outCt := conv.Foward(cipherInput)

	outFloat := ciphertextToFloat(outCt, cc)

	FloatToTxt("afterConv1.txt", outFloat)

}
func RotIndexToGaloisElements(input []int, context *customContext) *ckks.Evaluator {
	var galElements []uint64

	for _, rotIndex := range input {
		galElements = append(galElements, context.Params.GaloisElement(rotIndex))
	}
	galKeys := context.Kgen.GenGaloisKeysNew(galElements, context.Sk)

	newEvaluator := ckks.NewEvaluator(context.Params, rlwe.NewMemEvaluationKeySet(context.Kgen.GenRelinearizationKeyNew(context.Sk), galKeys...))

	return newEvaluator
}

// func inputToCipher(data [3072]byte) {
// 	slots := 32768
// 	//Check Size
// 	one_data_size := 4096
// 	data_copy_num := 8
// 	if one_data_size*data_copy_num != slots {
// 		fmt.Println("Error : In data_encrpytion(), You set encrypted data size to ", one_data_size*data_copy_num, ", but slot size is ", slots)
// 	}

// 	//data normalization (for cifar-10 data)
// 	means := []float64{0.4914, 0.4822, 0.4465}
// 	stds := []float64{0.2023, 0.1994, 0.2010}
// 	var vectorData [3072]float64
// 	for i := 0; i < 3072; i++ {
// 		rgb := i / 1024
// 		vectorData[i] = ((float64(data[i]) / 255.0) - means[rgb]) / stds[rgb]
// 	}

// 	// Copy paste vector
// 	ctLenFloat := make([]float64, one_data_size*data_copy_num)
// 	for i := 0; i < one_data_size; i++ {
// 		for j := 0; j < data_copy_num; j++ {
// 			if i >= len(vectorData) {
// 				ctLenFloat[i+one_data_size*j] = 0
// 			} else {
// 				ctLenFloat[i+one_data_size*j] = vectorData[i]
// 			}
// 		}
// 	}

// 	FloatToTxt("input.txt", ctLenFloat)

// }
func getCifar10() []CIFAR10Image {
	cifar10Filename := "cifar-10-batches-bin/data_batch_1.bin"

	file, err := os.Open(cifar10Filename)
	if err != nil {
		fmt.Println("Error: Cannot open file", cifar10Filename)
		return nil
	}
	defer file.Close()

	var images []CIFAR10Image

	for {
		var image CIFAR10Image
		err := binary.Read(file, binary.BigEndian, &image)
		if err != nil {
			break // 파일 끝에 도달하면 종료합니다.
		}
		images = append(images, image)
	}

	// 로딩된 데이터 확인 (예시로 첫 번째 이미지 출력)
	if len(images) > 0 {
		firstImage := images[0]
		fmt.Println("Label:", firstImage.Label)
		fmt.Println("Total #:", len(images))
	}
	return images
}

func setCKKSEnv() *customContext {
	context := new(customContext)
	// context.Params, _ = ckks.NewParametersFromLiteral(ckks.ParametersLiteral{
	// 	LogN:            16,
	// 	LogQ:            []int{49, 40, 40, 40, 40, 40, 40, 40, 40, 40, 40, 40, 40, 40, 40},
	// 	LogP:            []int{49, 49, 49},
	// 	LogDefaultScale: 40,
	// })

	context.Params, _ = ckks.NewParametersFromLiteral(ckks.ParametersLiteral{
		LogN: 16,
		LogQ: []int{51, 46, 46, 46, 46, 46, 46, 46, 46, 46, 46, 46, 46, 46, 46,
			46, 46, 46, 46, 46, 46, 46, 46, 46, 46}, //24개
		LogP:            []int{60, 60, 60, 60, 60}, //5개
		LogDefaultScale: 46,
	})

	context.Kgen = ckks.NewKeyGenerator(context.Params)

	context.Sk, context.Pk = context.Kgen.GenKeyPairNew()

	context.Encoder = ckks.NewEncoder(context.Params)

	context.EncryptorPk = ckks.NewEncryptor(context.Params, context.Pk)

	context.EncryptorSk = ckks.NewEncryptor(context.Params, context.Sk)

	context.Decryptor = ckks.NewDecryptor(context.Params, context.Sk)

	galElements := []uint64{context.Params.GaloisElement(2)}
	galKeys := context.Kgen.GenGaloisKeysNew(galElements, context.Sk)

	context.Evaluator = ckks.NewEvaluator(context.Params, rlwe.NewMemEvaluationKeySet(context.Kgen.GenRelinearizationKeyNew(context.Sk), galKeys...))

	// pt := ckks.NewPlaintext(context.Params, context.Params.MaxLevel())
	// ct := ckks.NewCiphertext(context.Params, 1, pt.Level())
	// ct2 := ckks.NewCiphertext(context.Params, 1, pt.Level())

	// float := makeRandomFloat(32768)
	// sample1DArray(float, 0, 10)
	// context.Encoder.Encode(float, pt)

	// context.EncryptorSk.Encrypt(pt, ct)
	// err := context.Evaluator.Rotate(ct, -1, ct2)
	// if err != nil {
	// 	fmt.Println(err)
	// }

	// context.Decryptor.Decrypt(ct2, pt)
	// context.Encoder.Decode(pt, float)
	// sample1DArray(float, 0, 10)

	return context
}
func resnetInferenceTest(layer int, cc *customContext) {
	//get Float
	exInput := txtToFloat("true_logs/layer0_input_data.txt")

	//Make it to ciphertext
	var copyInput []float64
	for i := 0; i < 8; i++ {
		for j := 0; j < 4096; j++ {
			if j < len(exInput) {
				copyInput = append(copyInput, exInput[j])
			} else {
				copyInput = append(copyInput, 0)
			}
		}
	}

	cipherInput := floatToCiphertextLevel(copyInput, 6, cc.Params, cc.Encoder, cc.EncryptorSk)

	resnet20 := NewResnetCifar10(layer, cc.Evaluator, cc.Encoder, cc.Decryptor, cc.Params, cc.EncryptorSk, cc.Kgen, cc.Sk)

	//cipherOutput :=
	resnet20.Inference(cipherInput)

}
func resnetprevInferenceTest(layer int, cc *customContext) {
	//get Float
	exInput := txtToFloat("true_logs/layer0_input_data.txt")

	//Make it to ciphertext
	var copyInput []float64
	for i := 0; i < 8; i++ {
		for j := 0; j < 4096; j++ {
			if j < len(exInput) {
				copyInput = append(copyInput, exInput[j])
			} else {
				copyInput = append(copyInput, 0)
			}
		}
	}

	cipherInput := floatToCiphertextLevel(copyInput, 6, cc.Params, cc.Encoder, cc.EncryptorSk)

	resnet20 := NewprevResnetCifar10(layer, cc.Evaluator, cc.Encoder, cc.Decryptor, cc.Params, cc.EncryptorSk, cc.Kgen, cc.Sk)

	//cipherOutput :=
	resnet20.Inference(cipherInput)

}
func prevResnetInferenceTest(layer int, cc *customContext) {
	//get Float
	exInput := txtToFloat("true_logs/layer0_input_data.txt")

	//Make it to ciphertext
	var copyInput []float64
	for i := 0; i < 8; i++ {
		for j := 0; j < 4096; j++ {
			if j < len(exInput) {
				copyInput = append(copyInput, exInput[j])
			} else {
				copyInput = append(copyInput, 0)
			}
		}
	}

	cipherInput := floatToCiphertextLevel(copyInput, 6, cc.Params, cc.Encoder, cc.EncryptorSk)

	resnet20 := NewprevResnetCifar10(layer, cc.Evaluator, cc.Encoder, cc.Decryptor, cc.Params, cc.EncryptorSk, cc.Kgen, cc.Sk)

	//cipherOutput :=
	resnet20.Inference(cipherInput)

}
func reluTest(cc *customContext) {
	//Make input float data
	inputFloat := makeRandomFloat(cc.Params.MaxSlots())

	//CKKS relu
	inputCt := floatToCiphertext(inputFloat, cc.Params, cc.Encoder, cc.EncryptorSk)

	relu := mulParModules.NewRelu(cc.Evaluator, cc.Encoder, cc.Decryptor, cc.EncryptorSk, cc.Params)

	outputCt := relu.Foward(inputCt)

	ckksOutputPlain := ckks.NewPlaintext(cc.Params, cc.Params.MaxLevel())
	ckksOutputFloat := make([]float64, cc.Params.MaxSlots())
	cc.Decryptor.Decrypt(outputCt, ckksOutputPlain)
	cc.Encoder.Decode(ckksOutputPlain, ckksOutputFloat)

	//Plaintext Relu
	outputFloat := plainRelu(inputFloat)

	//Error
	fmt.Println(euclideanDistance(ckksOutputFloat, outputFloat))

}

func logsCompare() {
	trueLogsPath := "./true_logs/"
	testLogsPath := "./myLogs/"
	var logsVector []string
	success := true

	trueLogFiles, err := ioutil.ReadDir(trueLogsPath)
	if err != nil {
		fmt.Println("Error reading true logs directory:", err)
		return
	}

	for _, trueLogFile := range trueLogFiles {

		currentFileName := trueLogFile.Name()
		// if currentFileName != "layer0_layerEnd.txt" {
		// 	continue
		// }
		if _, err := os.Stat(testLogsPath + currentFileName); err == nil {
			layerName := strings.Split(currentFileName, "_")[0]
			k := 1
			if layerName == "layer2" {
				k = 2
			} else if layerName == "layer3" {
				k = 4
			}

			trueLogs := txtToFloat(trueLogsPath + currentFileName)
			testLogs := txtToFloat(testLogsPath + currentFileName)

			var modifiedTrueLogs []float64
			if currentFileName != "AvgPoolEnd.txt" && currentFileName != "FcEnd.txt" {
				modifiedTrueLogs = packing(trueLogs, k)

				modifiedTrueLogs = copyPaste(modifiedTrueLogs, 32768/len(modifiedTrueLogs))
			} else if currentFileName == "AvgPoolEnd.txt" {
				modifiedTrueLogs = append0(trueLogs, 32768/8)
				modifiedTrueLogs = copyPaste(modifiedTrueLogs, 8)
			} else if currentFileName == "FcEnd.txt" {
				testLogs = testLogs[0:10]
				modifiedTrueLogs = trueLogs
			}

			// print(currentFileName)

			ed := euclideanDistance(modifiedTrueLogs, testLogs)
			if ed == -1 {
				success = false
				fmt.Println(currentFileName, "'s test Failed. Size above")
				continue
			}

			if ed < 1 {
				logsVector = append(logsVector, "Success! "+currentFileName+" : "+fmt.Sprintf("%f", ed))
			} else {
				logsVector = append(logsVector, "Failed "+currentFileName+" : "+fmt.Sprintf("%f", ed))
				success = false
			}
		}
	}

	sort.Strings(logsVector)

	fmt.Println("Euclidean Distance Logs Below")
	for _, str := range logsVector {
		fmt.Println(str)
	}
	fmt.Println()

	if success {
		fmt.Println("All logs are same! Success!!")
	}
}
func normalization(data [3072]byte) []float64 {
	means := []float64{0.4914, 0.4822, 0.4465}
	stds := []float64{0.2023, 0.1994, 0.2010}
	var vectorData [3072]float64
	for i := 0; i < 3072; i++ {
		rgb := i / 1024
		vectorData[i] = ((float64(data[i]) / 255.0) - means[rgb]) / stds[rgb]
	}
	return vectorData[:]
}
func max(floats []float64) int {
	max := floats[0] // 슬라이스의 첫 번째 요소를 초기 최댓값으로 설정합니다.
	answer := 0
	for index, value := range floats {
		if value > max {
			max = value // 새로운 최댓값을 찾으면 max에 할당합니다.
			answer = index
		}
	}
	return answer
}
func resnetInferenceForCifar10(layer int, cc *customContext, images []CIFAR10Image) {
	//Logs
	// Correct! for index :  937
	// correct / All :  934 / 938   99.57356076759062 %
	// Inference Time / Average Time :  33.87346969s / 34.098091481s

	resnet20 := NewResnetCifar10(layer, cc.Evaluator, cc.Encoder, cc.Decryptor, cc.Params, cc.EncryptorSk, cc.Kgen, cc.Sk)
	fmt.Println("Resnet Created!")
	correct := 0

	infAllTime := time.Now().Sub(time.Now())
	for imageNum := 0; imageNum < len(images); imageNum++ {

		exInput := normalization(images[imageNum].Data)
		label := int(images[imageNum].Label)

		//Make it to ciphertext
		var copyInput []float64
		for i := 0; i < 8; i++ {
			for j := 0; j < 4096; j++ {
				if j < len(exInput) {
					copyInput = append(copyInput, exInput[j])
				} else {
					copyInput = append(copyInput, 0)
				}
			}
		}

		cipherInput := floatToCiphertextLevel(copyInput, cc.Params.MaxLevel(), cc.Params, cc.Encoder, cc.EncryptorSk)
		start := time.Now()
		ctOut := resnet20.Inference(cipherInput)
		end := time.Now()
		infTime := end.Sub(start)

		infAllTime += infTime

		ptOut := cc.Decryptor.DecryptNew(ctOut)
		floatOut := make([]float64, cc.Params.MaxSlots())
		cc.Encoder.Decode(ptOut, floatOut)

		result := max(floatOut[0:10])

		if result == label {
			correct++
			fmt.Println("Correct! for index : ", imageNum)
		} else {
			fmt.Println("Wrong... for index : ", imageNum)
		}

		fmt.Println("correct / All : ", correct, "/", imageNum+1, " ", float64(correct)/float64(imageNum+1)*100.0, "%")
		fmt.Println("Inference Time / Average Time : ", infTime, "/", infAllTime/(time.Duration(imageNum+1)))
	}
}

func GalToEval(galKeys [][]*rlwe.GaloisKey, context *customContext) *ckks.Evaluator {
	var linGalKeys []*rlwe.GaloisKey

	for _, galKey := range galKeys {
		for _, g := range galKey {
			linGalKeys = append(linGalKeys, g)
		}
	}
	newEvaluator := ckks.NewEvaluator(context.Params, rlwe.NewMemEvaluationKeySet(context.Kgen.GenRelinearizationKeyNew(context.Sk), linGalKeys...))
	return newEvaluator
}

// Reorganize to -16384 ~ 16384. And remove repetitive elements.
func OrganizeRot(rotIndexes [][]int) [][]int {
	var result [][]int
	for level := 0; level < len(rotIndexes); level++ {
		//Reorganize
		rotateSets := make(map[int]bool)
		for _, each := range rotIndexes[level] {
			temp := each
			if temp > 16384 {
				temp = temp - 32768
			} else if temp < -16384 {
				temp = temp + 32768
			}
			rotateSets[temp] = true
		}
		//Change map to array
		var rotateArray []int
		for element := range rotateSets {
			if element != 0 {
				rotateArray = append(rotateArray, element)
			}
		}
		sort.Ints(rotateArray)
		//append to result
		result = append(result, rotateArray)
	}
	return result
}
func HierarchyKeyTest(layer int) {
	//Organize what kinds of key-level 0 keys needed.
	mulParRot, rotOptRot := RotKeyOrganize(layer)

	//For MulPar
	fmt.Println("===For MulPar!===")
	fmt.Println(Level1RotKeyNeededForInference(mulParRot))
	//For rotOptRot
	fmt.Println("===For RotOpt!===")
	fmt.Println(Level1RotKeyNeededForInference(rotOptRot))

}
func hoistSumTest(cc *customContext) {
	// Initial Time Test
	for ctLevel := 0; ctLevel <= cc.Params.MaxLevel(); ctLevel++ {
		fmt.Println("===== For ctLevel : ", ctLevel, " =====")
		// Make input float data
		inputFloat := makeRandomFloat(cc.Params.MaxSlots())
		ctIn := floatToCiphertextLevel(inputFloat, ctLevel, cc.Params, cc.Encoder, cc.EncryptorSk)
		rotIndex := []int{1, 2, 3, 4, 5, 6, 7}
		lenIndex := float64(len(rotIndex))
		newEv := rotIndexToGaloisEl(rotIndex, cc.Params, cc.Kgen, cc.Sk)

		// Get Hoist ratio of precomp and other and get mathematical solution (only first time)
		startTime := time.Now()
		ctOuts, _ := newEv.RotateHoistedNew(ctIn, rotIndex)
		for c := range ctOuts {
			newEv.Add(ctIn, c, ctIn)
		}
		endTime := time.Now()
		hoistTime := endTime.Sub(startTime).Milliseconds()

		startTime = time.Now()
		for i := 0; i < len(rotIndex); i++ {
			temp, _ := newEv.RotateNew(ctIn, rotIndex[i])
			newEv.Add(temp, ctIn, ctIn)
		}
		endTime = time.Now()
		originalTime := endTime.Sub(startTime).Milliseconds()

		fmt.Println(hoistTime, originalTime)
		precomp := float64(originalTime-hoistTime) / float64(lenIndex-1)
		fmt.Println("precomp : ", precomp)
		other := (float64(originalTime) - lenIndex*precomp) / lenIndex
		fmt.Println("other : ", other)

		fmt.Println(precomp + other*lenIndex)
		fmt.Println(precomp*lenIndex + other*lenIndex)

		// Get optimized solution
		// for length := 4; length < 32; length *= 2 {
		// 	fmt.Println("time	original hoist	Length")
		// 	fmt.Print(FindOptHoist(precomp, other, length))
		// 	fmt.Println("	", length)
		// }
		fmt.Print(FindOptHoist(precomp, other, 16))

		// // Run OptHoistSum
		// for size := 8; size <= 8; size *= 2 {
		// 	inputCt := floatToCiphertextLevel(inputFloat, 1, cc.Params, cc.Encoder, cc.EncryptorSk)
		// 	var rotIndexes []int
		// 	for i := 1; i < size; i++ {
		// 		rotIndexes = append(rotIndexes, i)
		// 	}

		// 	OptHoistSum(inputCt, rotIndexes, newEv)
		// }
	}

	//Simple Test
	// var inputFloat []float64
	// for i := 0; i < 32768; i++ {
	// 	j := i % 10
	// 	inputFloat = append(inputFloat, float64(j))
	// }

	// ctIn := floatToCiphertextLevel(inputFloat, 2, cc.Params, cc.Encoder, cc.EncryptorSk)
	// printCipherSample("output", ctIn, cc, 0, 10)
	// rotIndex := []int{1, -2, -4}
	// rotIndexes := []int{-6, -5, -4, -3 - 2, -1, 1, 2, 3, 4, 5, 6, 7}
	// newEv := rotIndexToGaloisEl(rotIndexes, cc.Params, cc.Kgen, cc.Sk)
	// startTime := time.Now()
	// ctOut := OptHoistSum(ctIn, rotIndex, newEv)
	// endTime := time.Now()

	// fmt.Println("Time : ", endTime.Sub(startTime))
	// printCipherSample("output", ctOut, cc, 0, 10)
}

// func algorTest() {
// 	//PrimMST test
// 	// graph := [][]int{
// 	// 	{0, 1, 3, 2, 2, 3},
// 	// 	{1, 0, 2, 2, 1, 2},
// 	// 	{3, 2, 0, 2, 1, 2},
// 	// 	{2, 2, 2, 0, 1, 2},
// 	// 	{2, 1, 1, 1, 0, 1},
// 	// 	{3, 2, 2, 2, 1, 0},
// 	// }
// 	// parent := PrimMST(graph)
// 	// for i := 1; i < len(graph); i++ {
// 	// 	println(parent[i], " - ", i, "\t", graph[i][parent[i]])
// 	// }
// 	// fmt.Println(parent)

// 	//Make Graph test

// 	eachInts := []int{1, 13, 16, 17, 19}
// 	move := []int{1, -1, 2, -2, 4, -4, 8, -8, 16, -16}

// 	nodes, graph, Hgraph := MakeGraph(eachInts, move)
// 	fmt.Println(nodes, graph)
// 	fmt.Println(Hgraph)

// }
func tempTimeTest(cc *customContext) {
	// Make input float data
	inputFloat := makeRandomFloat(cc.Params.MaxSlots())
	inputCt := floatToCiphertextLevel(inputFloat, 2, cc.Params, cc.Encoder, cc.EncryptorSk)

	inputPlain := ckks.NewPlaintext(cc.Params, cc.Params.MaxLevel())
	cc.Encoder.Encode(inputFloat, inputPlain)

	inputPlain2 := ckks.NewPlaintext(cc.Params, 2)
	cc.Encoder.Encode(inputFloat, inputPlain2)

	start := time.Now()
	cc.Evaluator.MulNew(inputCt, inputPlain)
	fmt.Println(time.Now().Sub(start))

	start = time.Now()
	cc.Evaluator.MulNew(inputCt, inputPlain2)
	fmt.Println(time.Now().Sub(start))

}
func generalKeyTest(cc *customContext) {

	hdnum := 4.0

	//register
	convIDs := []string{"CONV1", "CONV2", "CONV3s2", "CONV3", "CONV4s2", "CONV4"}
	maxDepth := []int{2, 2, 2, 2, 2, 2}

	mulPar := make([][]int, 3)
	rotOpt := make([][]int, 3)
	for index := 0; index < len(convIDs); index++ {
		mulParRot := mulParModules.MulParConvRegister(convIDs[index])
		rotOptRot := mulParModules.RotOptConvRegister(convIDs[index], maxDepth[index])

		for i := 0; i < 3; i++ {
			for _, each := range mulParRot[i] {
				mulPar[i] = append(mulPar[i], each)
			}

			for _, each := range rotOptRot[i] {
				rotOpt[i] = append(rotOpt[i], each)
			}
		}
	}

	for i := 0; i < 3; i++ {
		mulPar[i] = removeDuplicates(mulPar[i])
		rotOpt[i] = removeDuplicates(rotOpt[i])
	}

	fmt.Println(mulPar)
	fmt.Println(rotOpt)

	linmulPar := []int{}
	for _, each := range mulPar {
		for _, each1 := range each {
			linmulPar = append(linmulPar, each1)
		}
	}
	linmulPar = removeDuplicates(linmulPar)

	linrotOpt := []int{}
	for _, each := range rotOpt {
		for _, each1 := range each {
			linrotOpt = append(linrotOpt, each1)
		}
	}
	linrotOpt = removeDuplicates(linrotOpt)

	// With max Mult Level , 0 key level
	multMaxkey0 := NewGeneralKey(1, 0, cc.Params.MaxLevel(), &cc.Params)
	eachKeySize := multMaxkey0.GetKeySize()
	fmt.Println("==MulPar with max Mult Level, 0 key level==")
	fmt.Println(float64(eachKeySize*len(linmulPar))/1048576.0, "MB")
	fmt.Println("==RotOpt with max Mult Level, 0 key level==")
	fmt.Println(float64(eachKeySize*len(linrotOpt))/1048576.0, "MB")

	// multMaxkey0.PrintKeyInfo()
	// With max Mult Level , 1 key level
	lv1keys := []int{1, -1, 4, -4, 16, -16, 256, -256, 1024, -1024, 4096, -4096, 16384, -16384}
	multMaxkey1 := GenLevelUpKey(multMaxkey0, hdnum) //multMaxkey0.Hdnum
	eachKeySize = multMaxkey1.GetKeySize()

	// multMaxkey1.PrintKeyInfo()

	// fmt.Println("MulPar with max Mult Level, 1 key level")
	// fmt.Println(eachKeySize*len(lv1keys)/1048576, "MB")
	fmt.Println("==RotOpt with max Mult Level, 1 key level==")
	fmt.Println(float64(eachKeySize*len(lv1keys))/1048576.0, "MB")

	// With opt Mult Level , 1 key level
	// fmt.Println("MulPar with opt Mult Level, 1 key level")
	fmt.Println("==RotOpt with opt Mult Level, 0 key level==")
	mult2key0 := NewGeneralKey(1, 0, 2, &cc.Params)
	eachKeySize = mult2key0.GetKeySize()
	fmt.Println(float64(eachKeySize*len(linrotOpt))/1048576.0, "MB")

	//Final. Opt Mult Level, 1 key level
	fmt.Println("==RotOpt with opt Mult Level, 1 key level==")
	mult2key1 := GenLevelUpKey(mult2key0, hdnum)
	eachKeySize = mult2key1.GetKeySize()
	fmt.Println(float64(eachKeySize*len(linrotOpt))/1048576.0, "MB")

	// lv0key := NewGeneralKey(1, 0, 2, &cc.Params)
	// fmt.Println(lv0key.GetKeySize(), "byte")
	// fmt.Println(lv0key)

	// lv1key := GenLevelUpKey(lv0key, lv0key.Hdnum)
	// fmt.Println(lv1key.GetKeySize(), "byte")
	// fmt.Println(lv1key)

}
func getBluePrint() {
	convIDs := []string{"CONV1", "CONV2", "CONV3s2", "CONV3", "CONV4s2", "CONV4"}

	for index := 0; index < len(convIDs); index++ {
		convMap, _, _ := mulParModules.GetConvMap(convIDs[index], 3)
		rotSumBP := make([][]int, 1)
		rotSumBP[0] = []int{0}
		crossSumBP := make([]int, 0)
		for d := 1; d < len(convMap); d++ {

			if convMap[d][0] == 3 {
				crossSumBP = append(crossSumBP, convMap[d][1])
				crossSumBP = append(crossSumBP, 0)
				crossSumBP = append(crossSumBP, convMap[d][2:]...)
				break
			} else {
				rotSumBP = append(rotSumBP, convMap[d])
			}

		}
		rotSumBP[0][0] = len(rotSumBP) - 1

		fmt.Print("[")
		for _, row := range rotSumBP {
			fmt.Print("[")
			for i, val := range row {
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Printf("%d", val)
			}
			fmt.Print("]")
		}
		fmt.Println("]")

		fmt.Print("[")
		for i, val := range crossSumBP {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Printf("%d", val)
		}
		fmt.Print("]")
		fmt.Println()
		fmt.Println()
		// fmt.Println(rotSumBP, crossSumBP)
	}

}
func rotOptConvAccuracyTestForAllConv(layer int, context *customContext) {
	convIDs := []string{"CONV1", "CONV2", "CONV3s2", "CONV3", "CONV4s2", "CONV4"}
	maxConvDepth := []int{2, 4, 5, 4, 5, 4}

	startDepth := 2
	for index := 0; index < len(convIDs); index++ {
		for convDepth := startDepth; convDepth < maxConvDepth[index]+1; convDepth++ { //원래 depth:=2
			rotOptConvAccuracyTest(layer, context, convIDs[index], convDepth, convDepth)
		}
	}

}

func mulParConvAccuracyTestForAllConv(layer int, context *customContext) {
	convIDs := []string{"CONV1", "CONV2", "CONV3s2", "CONV3", "CONV4s2", "CONV4"}

	for index := 0; index < len(convIDs); index++ {
		mulParConvAccuracyTest(layer, context, convIDs[index], 2)
	}
}
func main() {

	//CKKS settings
	context := setCKKSEnv()

	//Resnet Setting
	// images := getCifar10()
	layer := 20

	// conv1Test(context)
	// resnetInferenceTest(layer, context) //You have to enable myLogsSave codes in resnet.go to use this function.
	// resnetprevInferenceTest(layer, context)
	// resnetInferenceForCifar10(layer, context, images)
	// logsCompare()

	// Basic Operation Tests
	// basicTest()
	// rotationTimeTest(context)
	// addMultTimeTest(context)
	// tempTimeTest(context)

	// resnet operation tests
	// avgPoolTest(context)
	// fullyConnectedTest(layer, context)
	// rotOptDownSamplingTest(context)
	// mulParDownSamplingTest(context)
	// reluTest(context)

	// Convolution Tests
	// rotOptConvTest(layer, context, 2)
	// mulParConvTest(layer, context, 2)
	rotOptConvAccuracyTestForAllConv(layer, context)
	mulParConvAccuracyTestForAllConv(layer, context)

	//Hoist sum test
	// hoistSumTest(context)

	//Server - Client RotKey Test
	// HierarchyKeyTest(layer)
	// generalKeyTest(context)

	// Print Blue Print
	// getBluePrint()
}
