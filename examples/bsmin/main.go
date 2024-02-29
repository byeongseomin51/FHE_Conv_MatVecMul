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
	newEvaluator := rotIndexToGaloisElements(rot, cc)

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
	newEvaluator := rotIndexToGaloisElements(rot, cc)

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

func downSamplingTest(cc *customContext) {
	//register
	rot := mulParModules.RotOptDSRegister()

	//rot register
	newEvaluator := rotIndexToGaloisElements(rot, cc)

	//make avgPooling instance
	ds16 := mulParModules.NewRotOptDS(16, newEvaluator, cc.Encoder, cc.Params)
	ds32 := mulParModules.NewRotOptDS(32, newEvaluator, cc.Encoder, cc.Params)

	//Make input float data
	inputFloat := makeRandomFloat(cc.Params.MaxSlots())

	//Encryption
	inputCt := floatToCiphertext(inputFloat, cc.Params, cc.Encoder, cc.EncryptorSk)
	///////////
	//Timer start
	startTime := time.Now()

	//AvgPooling Foward
	outputCt16 := ds16.Foward(inputCt)

	//Timer end
	endTime := time.Now()

	//Print Elapsed Time
	fmt.Printf("Time(16) : %v \n", endTime.Sub(startTime))
	//////////
	//Timer start
	startTime = time.Now()

	//AvgPooling Foward
	outputCt32 := ds32.Foward(inputCt)

	//Timer end
	endTime = time.Now()

	//Print Elapsed Time
	fmt.Printf("Time(32) : %v \n", endTime.Sub(startTime))

	//Decryption
	outputFloat16 := ciphertextToFloat(outputCt16, cc)
	outputFloat32 := ciphertextToFloat(outputCt32, cc)

	//Test
	fmt.Println("==16==")
	count01num(outputFloat16)
	fmt.Println("\n==32==")
	count01num(outputFloat32)
}

func getConvTestTxtPath(convID string) string {
	if convID == "CONV1" {
		return "conv1_weight.txt"
	} else if convID == "CONV2" {
		return "layer1_0_conv1_weight.txt"
	} else if convID == "CONV3s2" {
		return "layer2_0_conv1_weight.txt"
	} else if convID == "CONV3" {
		return "layer2_0_conv2_weight.txt"
	} else if convID == "CONV4s2" {
		return "layer3_0_conv1_weight.txt"
	} else if convID == "CONV4" {
		return "layer3_0_conv2_weight.txt"
	}
	return ""
}

func getConvTestNum(convID string) []int {
	if convID == "CONV1" {
		return []int{0, 1}
	} else if convID == "CONV2" {
		return []int{0, 1}
	} else if convID == "CONV3s2" {
		return []int{0, 1}
	} else if convID == "CONV3" {
		return []int{0, 2}
	} else if convID == "CONV4s2" {
		return []int{0, 1}
	} else if convID == "CONV4" {
		return []int{0, 2}
	}
	return []int{}
}
func mulParConvTest(layerNum int, cc *customContext) {
	// mulParModules.MakeTxtRotOptConvWeight()
	// mulParModules.MakeTxtRotOptConvFilter()
	convIDs := []string{"CONV1", "CONV2", "CONV3s2", "CONV3", "CONV4s2", "CONV4"}
	maxDepth := []int{2, 2, 2, 2, 2, 2}

	for index := 0; index < len(convIDs); index++ {
		for depth := 2; depth < maxDepth[index]+1; depth++ {
			convID := convIDs[index]
			fmt.Printf("convID : %s, Depth : %v\n", convID, depth)

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

			kernel_filePath := "mulParModules/precomputed/resnetPtParam/" + strconv.Itoa(layerNum) + "/" + getConvTestTxtPath(convID)

			flattenOriginal := flatten(realConv(cf.InputDataWidth, cf.InputDataHeight, cf.InputDataChannel, cf.KernelSize, cf.KernelNumber, cf.Stride, inputRandomVector, kernelTxtToVector(kernel_filePath)))
			flattenOriginal = copyPaste(flattenOriginal, cf.AfterCopy)
			if len(flattenOriginal) != 32768 {
				fmt.Println("You have set wrong parameter!")
			}

			////CKKS convolution////
			plain := ckks.NewPlaintext(cc.Params, depth)
			cc.Encoder.Encode(flattenInputData, plain)
			inputCt, _ := cc.EncryptorSk.EncryptNew(plain)

			//register
			rot := mulParModules.MulParConvRegister(convID)
			// fmt.Println(rot)

			// for i := 0; i < 3; i++ {
			// 	fmt.Println(len(rot[i]))
			// }

			//rot register
			newEvaluator := rotIndexToGaloisEl(int2dTo1d(rot), cc.Params, cc.Kgen, cc.Sk)

			//make rotOptConv instance
			conv := mulParModules.NewMulParConv(newEvaluator, cc.Encoder, cc.Decryptor, cc.Params, layerNum, convID, depth, getConvTestNum(convID)[0], getConvTestNum(convID)[1])
			//Timer start
			startTime := time.Now()

			//Conv Foward
			outputCt := conv.Foward(inputCt)

			//Timer end
			endTime := time.Now()

			//Print Elapsed Time
			fmt.Printf("Time : %v \n", endTime.Sub(startTime))

			//Decryption
			outputFloat := ciphertextToFloat(outputCt, cc)

			// fmt.Println(euclideanDistance(unpacking(outputFloat, cf.InputDataWidth/cf.Stride, cf.K*cf.Stride), flattenOriginal))
			euclideanDistance(unpacking(outputFloat, cf.InputDataWidth/cf.Stride, cf.K*cf.Stride), flattenOriginal)

			// // sample1DArray(outputFloat, 0, 32768)
			// sample1DArray(outputFloat, 0, 32768)
		}
	}
}
func rotOptConvTest(layerNum int, cc *customContext) {
	// mulParModules.MakeTxtRotOptConvWeight()
	// mulParModules.MakeTxtRotOptConvFilter()
	convIDs := []string{"CONV1", "CONV2", "CONV3s2", "CONV3", "CONV4s2", "CONV4"}
	// convIDs := []string{"CONV3s2"}
	maxDepth := []int{2, 4, 5, 4, 5, 4}
	// maxDepth := []int{5}
	// maxDepth := []int{2, 2, 2, 2, 2, 2}

	startDepth := 2

	for index := 0; index < len(convIDs); index++ {
		for depth := startDepth; depth < maxDepth[index]+1; depth++ { //원래 depth:=2
			convID := convIDs[index]

			fmt.Printf("=== convID : %s, Depth : %v === \n", convID, depth)

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

			kernel_filePath := "mulParModules/precomputed/resnetPtParam/" + strconv.Itoa(layerNum) + "/" + getConvTestTxtPath(convID)

			flattenOriginal := flatten(realConv(cf.InputDataWidth, cf.InputDataHeight, cf.InputDataChannel, cf.KernelSize, cf.KernelNumber, cf.Stride, inputRandomVector, kernelTxtToVector(kernel_filePath)))
			flattenOriginal = copyPaste(flattenOriginal, cf.AfterCopy)
			if len(flattenOriginal) != 32768 {
				fmt.Println("You have set wrong parameter!")
			}

			////CKKS convolution////
			plain := ckks.NewPlaintext(cc.Params, depth)
			cc.Encoder.Encode(flattenInputData, plain)
			inputCt, _ := cc.EncryptorSk.EncryptNew(plain)

			//register
			rots := mulParModules.RotOptConvRegister(convID, depth)

			fmt.Println(rots)

			//rot register
			newEvaluator := rotIndexToGaloisEl(int2dTo1d(rots), cc.Params, cc.Kgen, cc.Sk)

			//make rotOptConv instance
			conv := mulParModules.NewrotOptConv(newEvaluator, cc.Encoder, cc.Decryptor, cc.Params, layerNum, convID, depth, getConvTestNum(convID)[0], getConvTestNum(convID)[1])

			//Timer start
			startTime := time.Now()

			//Conv Foward
			outputCt := conv.Foward(inputCt)

			//Timer end
			endTime := time.Now()

			//Print Elapsed Time
			fmt.Printf("Time : %v \n", endTime.Sub(startTime))

			//Decryption
			outputFloat := ciphertextToFloat(outputCt, cc)

			// fmt.Println(euclideanDistance(unpacking(outputFloat, cf.InputDataWidth/cf.Stride, cf.K*cf.Stride), flattenOriginal))
			euclideanDistance(unpacking(outputFloat, cf.InputDataWidth/cf.Stride, cf.K*cf.Stride), flattenOriginal)

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
	newEvaluator := rotIndexToGaloisElements(int2dTo1d(rots), cc)

	conv := mulParModules.NewMulParConv(newEvaluator, cc.Encoder, cc.Decryptor, cc.Params, 20, "CONV1", 2, 0, 1)

	outCt := conv.Foward(cipherInput)

	outFloat := ciphertextToFloat(outCt, cc)

	FloatToTxt("afterConv1.txt", outFloat)

}
func rotIndexToGaloisElements(input []int, context *customContext) *ckks.Evaluator {
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
		LogN:            16,
		LogQ:            []int{51, 46, 46, 46, 46, 46, 46, 46, 46, 46, 46, 46, 46, 46, 46},
		LogP:            []int{60, 60, 60},
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
func resnetInferenceForCifar10(layer int, cc *customContext) {
	// //get Float

	// //Make it to ciphertext
	// var copyInput []float64
	// for i := 0; i < 8; i++ {
	// 	for j := 0; j < 4096; j++ {
	// 		if j < len(exInput) {
	// 			copyInput = append(copyInput, exInput[j])
	// 		} else {
	// 			copyInput = append(copyInput, 0)
	// 		}
	// 	}
	// }

	// cipherInput := floatToCiphertextLevel(copyInput, 6, cc.Params, cc.Encoder, cc.EncryptorSk)

	// resnet20 := NewResnetCifar10(layer, cc.Evaluator, cc.Encoder, cc.Decryptor, cc.Params, cc.EncryptorSk, cc.Kgen, cc.Sk)

	// //cipherOutput :=
	// resnet20.Inference(cipherInput)

}

func serverTest() {
	// clientContext := setCKKSEnv()

	// layer := 20

	// serverResnet20 := NewResnetCifar10(layer, clientContext.Evaluator, clientContext.Encoder, clientContext.Decryptor, clientContext.Params, clientContext.EncryptorSk, clientContext.Kgen, clientContext.Sk)

	// requireInit := serverResnet20.ClientRotKeyNeeded()

	// //Do initRotKeyGen base on require init

	// serverResnet20.GiveRotKey()

	// //serverResnet20.Inference()

}
func main() {

	// images := getCifar10()
	context := setCKKSEnv()
	layer := 20

	// conv1Test(context)
	// resnetInferenceTest(layer, context)
	resnetprevInferenceTest(layer, context)
	// resnetInferenceForCifar10(layer, context)
	logsCompare()

	// Basic Operation Tests
	// basicTest()
	// rotationTimeTest(context)
	// addMultTimeTest(context)

	// resnet operation tests
	// avgPoolTest(context)
	// fullyConnectedTest(layer, context)
	// downSamplingTest(context)
	// reluTest(context)

	// Convolution Tests
	// rotOptConvTest(layer, context)
	// mulParConvTest(layer, context)

	//Server - Client Test
	// serverTest()

}

// kernel 문제...? 아니면 rotation 문제.,?
// Euclidean Distance Logs Below
// Failed AvgPoolEnd.txt : 27.321475
// Failed FcEnd.txt : 40.935761
// Failed layer2_0_bn1.txt : 337.759744
// Failed layer2_0_bn2.txt : 283.547389
// Failed layer2_1_bn1.txt : 132.513293
// Failed layer2_1_bn2.txt : 99.903144
// Failed layer2_2_bn1.txt : 144.418705
// Failed layer2_2_bn2.txt : 105.745933
// Failed layer2_layerEnd.txt : 189.854234
// Failed layer3_0_bn1.txt : 214.484531
// Failed layer3_0_bn2.txt : 289.551886
// Failed layer3_1_bn1.txt : 148.951862
// Failed layer3_1_bn2.txt : 175.963691
// Failed layer3_2_bn1.txt : 150.383789
// Failed layer3_2_bn2.txt : 466.998735
// Failed layer3_layerEnd.txt : 350.984730
// Success! layer0_layerEnd.txt : 0.000014
// Success! layer1_0_bn1.txt : 0.000031
// Success! layer1_0_bn2.txt : 0.000024
// Success! layer1_1_bn1.txt : 0.000039
// Success! layer1_1_bn2.txt : 0.000030
// Success! layer1_2_bn1.txt : 0.000054
// Success! layer1_2_bn2.txt : 0.000036
// Success! layer1_layerEnd.txt : 0.000051
