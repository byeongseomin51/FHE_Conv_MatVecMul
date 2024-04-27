package main

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"rotOptResnet/mulParModules"
	"sort"
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

func parfullyConnectedAccuracyTest(layerNum int, cc *customContext) {
	startLevel := 2
	endLevel := cc.Params.MaxLevel()
	// endLevel := 2
	//register
	rot := mulParModules.ParFCRegister()

	//rot register
	newEvaluator := RotIndexToGaloisElements(rot, cc)

	//make avgPooling instance
	fc := mulParModules.NewParFC(newEvaluator, cc.Encoder, cc.Params, layerNum)

	//Make input float data
	temp := txtToFloat("true_logs/AvgPoolEnd.txt")
	trueInputFloat := make([]float64, 32768)
	for i := 0; i < len(temp); i++ {
		for par := 0; par < 8; par++ {
			trueInputFloat[i+4096*par] = temp[i]
		}
	}

	//Make output float data
	trueOutputFloat := txtToFloat("true_logs/FcEnd.txt")

	var outputCt *rlwe.Ciphertext
	for level := startLevel; level <= endLevel; level++ {
		// Encryption
		inputCt := floatToCiphertextLevel(trueInputFloat, level, cc.Params, cc.Encoder, cc.EncryptorSk)

		// Timer start
		startTime := time.Now()

		// AvgPooling Foward
		outputCt = fc.Foward(inputCt)

		// Timer end
		endTime := time.Now()

		// Print Elapsed Time
		fmt.Printf("%v %v \n", level, TimeDurToFloatSec(endTime.Sub(startTime)))

	}

	//Decryption
	outputFloat := ciphertextToFloat(outputCt, cc)

	fmt.Println("Accuracy : ", euclideanDistance(outputFloat[0:10], trueOutputFloat))
}

func mulParfullyConnectedAccuracyTest(layerNum int, cc *customContext) {
	startLevel := 2
	endLevel := cc.Params.MaxLevel()
	//register
	rot := mulParModules.MulParFCRegister()

	//rot register
	newEvaluator := RotIndexToGaloisElements(rot, cc)

	//make avgPooling instance
	fc := mulParModules.NewMulParFC(newEvaluator, cc.Encoder, cc.Params, layerNum)

	//Make input float data
	temp := txtToFloat("true_logs/AvgPoolEnd.txt")
	trueInputFloat := make([]float64, 32768)
	for i := 0; i < len(temp); i++ {
		for par := 0; par < 8; par++ {
			trueInputFloat[i+4096*par] = temp[i]
		}
	}

	//Make output float data
	trueOutputFloat := txtToFloat("true_logs/FcEnd.txt")

	var outputCt *rlwe.Ciphertext
	for level := startLevel; level <= endLevel; level++ {
		// Encryption
		inputCt := floatToCiphertextLevel(trueInputFloat, level, cc.Params, cc.Encoder, cc.EncryptorSk)

		// Timer start
		startTime := time.Now()

		// AvgPooling Foward
		outputCt = fc.Foward(inputCt)

		// Timer end
		endTime := time.Now()

		// Print Elapsed Time
		fmt.Printf("%v %v \n", level, TimeDurToFloatSec(endTime.Sub(startTime)))

	}

	//Decryption
	outputFloat := ciphertextToFloat(outputCt, cc)

	fmt.Println("Accuracy : ", euclideanDistance(outputFloat[0:10], trueOutputFloat))

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
	var outputCt16 *rlwe.Ciphertext
	var outputCt32 *rlwe.Ciphertext
	for level := 2; level <= cc.Params.MaxLevel(); level++ {
		// Encryption
		inputCt := floatToCiphertextLevel(inputFloat, level, cc.Params, cc.Encoder, cc.EncryptorSk)
		// /////////
		// Timer start
		startTime := time.Now()

		// AvgPooling Foward
		outputCt16 = ds16.Foward(inputCt)

		// Timer end
		endTime := time.Now()

		// Print Elapsed Time
		// fmt.Printf("%v Time(16) : %v \n", level,TimeDurToFloatSec(endTime.Sub(startTime)))
		// ////////
		// Timer start
		startTime = time.Now()

		// AvgPooling Foward
		outputCt32 = ds32.Foward(inputCt)

		// Timer end
		endTime = time.Now()

		// Print Elapsed Time
		fmt.Printf("%v Time(32) : %v \n", level, TimeDurToFloatSec(endTime.Sub(startTime)))
	}

	// 	//Decryption
	outputFloat16 := ciphertextToFloat(outputCt16, cc)
	outputFloat32 := ciphertextToFloat(outputCt32, cc)

	// //Test
	fmt.Println("==16==")
	Count01num(outputFloat16)
	fmt.Println("\n==32==")
	Count01num(outputFloat32)
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
	var outputCt16 *rlwe.Ciphertext
	var outputCt32 *rlwe.Ciphertext
	for level := 2; level <= cc.Params.MaxLevel(); level++ {
		// Encryption
		inputCt := floatToCiphertextLevel(inputFloat, level, cc.Params, cc.Encoder, cc.EncryptorSk)
		// /////////
		// Timer start
		startTime := time.Now()

		// AvgPooling Foward
		outputCt16 = ds16.Foward(inputCt)

		// Timer end
		endTime := time.Now()

		// Print Elapsed Time
		fmt.Printf("%v Time(16) : %v \n", level, TimeDurToFloatSec(endTime.Sub(startTime)))
		// ////////
		// Timer start
		startTime = time.Now()

		// AvgPooling Foward
		outputCt32 = ds32.Foward(inputCt)

		// Timer end
		endTime = time.Now()

		// Print Elapsed Time
		// fmt.Printf("%v Time(32) : %v \n", level, TimeDurToFloatSec(endTime.Sub(startTime)))
	}

	// 	//Decryption
	// ciphertextToFloat(outputCt16, cc)
	// ciphertextToFloat(outputCt32, cc)
	outputFloat16 := ciphertextToFloat(outputCt16, cc)
	outputFloat32 := ciphertextToFloat(outputCt32, cc)

	// //Test
	fmt.Println("==16==")
	Count01num(outputFloat16)
	fmt.Println("\n==32==")
	Count01num(outputFloat32)
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

func rotOptConvTimeTest(layerNum int, cc *customContext) {
	fmt.Println("RotOptConvTimeTest started!")

	//For log
	currentTime := time.Now()
	logFileName := "RotOptLog_" + currentTime.Format("2006-01-02_15-04-05") + ".txt"
	logFile, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.SetFlags(0)

	// mulParModules.MakeTxtRotOptConvWeight()
	// mulParModules.MakeTxtRotOptConvFilter()
	// convIDs := []string{"CONV1", "CONV2", "CONV3s2", "CONV3", "CONV4s2", "CONV4"}
	convIDs := []string{"CONV4s2", "CONV4"}
	// convIDs := []string{"CONV4"}
	// maxDepth := []int{3}
	// maxDepth := []int{4}
	// maxDepth := []int{2, 3, 3, 3, 3, 3}
	// maxDepth := []int{2, 2, 2, 2, 2, 2}
	maxDepth := []int{2, 2}
	// maxDepth := []int{4}

	//Set min depth
	// startDepth := 2
	startDepth := 2

	//Set iter
	iter := 10

	// minStartCipherLevel := 2
	minStartCipherLevel := 2
	maxStartCipherLevel := cc.Params.MaxLevel()

	for index := 0; index < len(convIDs); index++ {
		for depth := startDepth; depth < maxDepth[index]+1; depth++ { //원래 depth:=2
			convID := convIDs[index]

			inputRandomVector := makeRandomFloat(cc.Params.MaxSlots())

			//register
			rots := mulParModules.RotOptConvRegister(convID, depth)
			// for _, r := range rots {
			// 	fmt.Println(len(r), r)
			// }

			//rot register
			newEvaluator := rotIndexToGaloisEl(int2dTo1d(rots), cc.Params, cc.Kgen, cc.Sk)

			//make rotOptConv instance
			conv := mulParModules.NewrotOptConv(newEvaluator, cc.Encoder, cc.Decryptor, cc.Params, layerNum, convID, depth, getConvTestNum(convID)[0], getConvTestNum(convID)[1])

			// fmt.Printf("=== convID : %s, Depth : %v, CipherLevel : %v ~ %v, iter : %v === \n", convID, depth, minStartCipherLevel, maxStartCipherLevel, iter)
			log.Printf("=== convID : %s, Depth : %v, CipherLevel : %v ~ %v, iter : %v === \n", convID, depth, minStartCipherLevel, maxStartCipherLevel, iter)

			for startCipherLevel := Max(minStartCipherLevel, depth); startCipherLevel <= maxStartCipherLevel; startCipherLevel++ {

				plain := ckks.NewPlaintext(cc.Params, startCipherLevel)
				cc.Encoder.Encode(inputRandomVector, plain)
				inputCt, _ := cc.EncryptorSk.EncryptNew(plain)

				//Timer start
				startTime := time.Now()

				//Conv Foward
				for i := 0; i < iter; i++ {
					conv.Foward(inputCt)
				}

				//Timer end
				endTime := time.Now()

				//Print Elapsed Time
				time := float64((endTime.Sub(startTime) / time.Duration(iter)).Nanoseconds()) / 1e9
				// fmt.Printf("%v %v \n", startCipherLevel, time)
				log.Printf("%v %v \n", startCipherLevel, time)
				// fmt.Printf("Time : %v \n", endTime.Sub(startTime)/time.Duration(iter))
				// fmt.Printf("%v) \n", endTime.Sub(startTime)/time.Duration(iter))
			}
		}
	}
}
func mulParConvTimeTest(layerNum int, cc *customContext) {
	fmt.Println("MulParConvTimeTest")
	// mulParModules.MakeTxtMulParConvWeight()
	// mulParModules.MakeTxtMulParConvFilter()
	convIDs := []string{"CONV1", "CONV2", "CONV3s2", "CONV3", "CONV4s2", "CONV4"}
	// convIDs := []string{"CONV3", "CONV4s2"}
	// convIDs := []string{"CONV2"}

	//For log
	currentTime := time.Now()
	logFileName := "MulParLog_" + currentTime.Format("2006-01-02_15-04-05") + ".txt"
	logFile, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.SetFlags(0)

	//Set iter
	iter := 10

	minStartCipherLevel := 2
	maxStartCipherLevel := cc.Params.MaxLevel()

	for index := 0; index < len(convIDs); index++ {
		convID := convIDs[index]

		inputRandomVector := makeRandomFloat(cc.Params.MaxSlots())

		//register
		rots := mulParModules.MulParConvRegister(convID)
		// for _, r := range rots {
		// 	fmt.Println(len(r), r)
		// }

		//rot register
		newEvaluator := rotIndexToGaloisEl(int2dTo1d(rots), cc.Params, cc.Kgen, cc.Sk)

		//make mulParConv instance
		conv := mulParModules.NewMulParConv(newEvaluator, cc.Encoder, cc.Decryptor, cc.Params, layerNum, convID, getConvTestNum(convID)[0], getConvTestNum(convID)[1])

		log.Printf("=== convID : %s, Depth : %v, CipherLevel : %v ~ %v, iter : %v === \n", convID, 2, minStartCipherLevel, maxStartCipherLevel, iter)
		// fmt.Printf("=== convID : %s, Depth : %v, CipherLevel : %v ~ %v, iter : %v === \n", convID, 2, minStartCipherLevel, maxStartCipherLevel, iter)

		for startCipherLevel := minStartCipherLevel; startCipherLevel <= maxStartCipherLevel; startCipherLevel++ {

			plain := ckks.NewPlaintext(cc.Params, startCipherLevel)
			cc.Encoder.Encode(inputRandomVector, plain)
			inputCt, _ := cc.EncryptorSk.EncryptNew(plain)

			//Timer start
			startTime := time.Now()

			//Conv Foward
			for i := 0; i < iter; i++ {
				conv.Foward(inputCt)
			}

			//Timer end
			endTime := time.Now()

			//Print Elapsed Time
			time := float64((endTime.Sub(startTime) / time.Duration(iter)).Nanoseconds()) / 1e9
			log.Printf("%v %v \n", startCipherLevel, time)
			// fmt.Printf("%v %v \n", startCipherLevel, time)

			// fmt.Printf("Time : %v \n", endTime.Sub(startTime)/time.Duration(iter))
			// fmt.Printf("%v) \n", endTime.Sub(startTime)/time.Duration(iter))
		}

	}
}
func mulParConvRotKeyTest(layerNum int) {
	fmt.Println("MulParConvTimeTest")
	// mulParModules.MakeTxtMulParConvWeight()
	// mulParModules.MakeTxtMulParConvFilter()
	convIDs := []string{"CONV1", "CONV2", "CONV3s2", "CONV3", "CONV4s2", "CONV4"}
	// convIDs := []string{"CONV3", "CONV4s2"}
	// convIDs := []string{"CONV2"}

	for index := 0; index < len(convIDs); index++ {
		//register
		rots := mulParModules.MulParConvRegister(convIDs[index])
		for _, i := range rots {
			fmt.Println(len(i))
		}
	}
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
	logsCompare()
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

	logsCompare()
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
func resnetInferenceForCifar10(layer int, cc *customContext) {
	images := getCifar10()

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
		convMap, _, _ := mulParModules.GetConvMap(convIDs[index], 5)
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
	convIDs := []string{"CONV2"}
	// convIDs := []string{"CONV1", "CONV2", "CONV3s2", "CONV3", "CONV4s2", "CONV4"}

	// maxConvDepth := []int{2, 4, 5, 4, 5, 4}
	maxConvDepth := []int{4}

	startDepth := 4
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
func basicOperationTimeTest(cc *customContext) {
	floats := makeRandomFloat(32768)

	rot := make([]int, 1)
	rot[0] = 1

	//rot register
	newEvaluator := RotIndexToGaloisElements(rot, cc)
	for i := 0; i <= cc.Params.MaxLevel(); i++ {
		cipher1 := floatToCiphertextLevel(floats, i, cc.Params, cc.Encoder, cc.EncryptorSk)
		start1 := time.Now()
		newEvaluator.Rotate(cipher1, 1, cipher1)
		end1 := time.Now()

		start2 := time.Now()
		newEvaluator.Add(cipher1, cipher1, cipher1)
		end2 := time.Now()

		newEvaluator.Mul(cipher1, cipher1, cipher1)
		start3 := time.Now()
		newEvaluator.Rescale(cipher1, cipher1)
		end3 := time.Now()
		fmt.Println(i, TimeDurToFloatMiliSec(end1.Sub(start1)), TimeDurToFloatMiliSec(end2.Sub(start2)), TimeDurToFloatMiliSec(end3.Sub(start3)))
	}

}

func bsgsMatVecMultAccuracyTest(cc *customContext) {
	nt := 32768

	for i := 16; i <= 512; i *= 2 {
		N := i
		fmt.Printf("=== Conevntional (BSGS diag mat(N*N)-vec(N*1) mul) method start! N : %v ===\n", N)
		A := getPrettyMatrix(N, N)
		B := getPrettyMatrix(N, 1)

		//answer
		answer := originalMatMul(A, B)

		//change B to ciphertext
		B1d := make2dTo1d(B)
		B1d = resize(B1d, nt)
		Bct := floatToCiphertextLevel(B1d, 2, cc.Params, cc.Encoder, cc.EncryptorSk)

		//start mat vec mul
		rot := mulParModules.BsgsDiagMatVecMulRegister(N)
		newEvaluator := RotIndexToGaloisElements(rot, cc)
		matVecMul := mulParModules.NewBsgsDiagMatVecMul(A, N, nt, newEvaluator, cc.Encoder, cc.Params)
		startTime := time.Now()
		BctOut := matVecMul.Foward(Bct)
		endTime := time.Now()
		outputFloat := ciphertextToFloat(BctOut, cc)

		fmt.Println("Time : ", TimeDurToFloatMiliSec(endTime.Sub(startTime)), " ms")
		fmt.Println("Accuracy : ", euclideanDistance(outputFloat[0:N], make2dTo1d(answer)))
		n1, n2 := mulParModules.FindBsgsSol(N)
		fmt.Println("Rotation :", n1+n2-1, ", Mul :", n1*n2, ", Add :", n1*n2+n1+1)
	}
}
func parBsgsMatVecMultAccuracyTest(cc *customContext) {
	nt := cc.Params.MaxSlots()
	pi := 1
	for i := 16; i <= 512; i *= 2 {
		N := i
		fmt.Printf("=== Proposed (Parallely BSGS diag mat(N*N)-vec(N*1) mul) method start! N : %v ===\n", N)
		A := getPrettyMatrix(N, N)
		B := getPrettyMatrix(N, 1)

		//answer
		answer := originalMatMul(A, B)

		//change B to ciphertext
		B1d := make2dTo1d(B)
		B1d = resize(B1d, nt)
		for i := 1; i < pi; i *= 2 {
			tempB := rotate(B1d, -(nt/pi)*i)
			B1d = add(tempB, B1d)
		}
		Bct := floatToCiphertextLevel(B1d, 2, cc.Params, cc.Encoder, cc.EncryptorSk)

		//start mat vec mul
		rot := mulParModules.ParBsgsDiagMatVecMulRegister(N, nt, pi)
		newEvaluator := RotIndexToGaloisElements(rot, cc)
		matVecMul := mulParModules.NewParBsgsDiagMatVecMul(A, N, nt, pi, newEvaluator, cc.Encoder, cc.Params)
		startTime := time.Now()
		BctOut := matVecMul.Foward(Bct)
		endTime := time.Now()
		outputFloat := ciphertextToFloat(BctOut, cc)

		fmt.Println("Time : ", TimeDurToFloatMiliSec(endTime.Sub(startTime)), " ms")
		fmt.Println("Accuracy : ", euclideanDistance(outputFloat[0:N], make2dTo1d(answer)))
		n1, n2 := mulParModules.FindParBsgsSol(N, nt, pi)
		fmt.Println("Rotation :", 2*int(math.Log2(float64(n2)))+(n1)-int(math.Log2(float64(pi))), ", Mul :", n1, ", Add :", n1+int(math.Log2(float64(n2)))+1)
	}
}
func main() {

	//CKKS settings
	context := setCKKSEnv()

	// Convolution Tests
	// rotOptConvAccuracyTestForAllConv(20, context)
	// mulParConvAccuracyTestForAllConv(20, context)
	// rotOptConvTimeTest(20, context)
	// mulParConvTimeTest(20, context)

	// Operation tests
	// avgPoolTest(context)
	// parfullyConnectedAccuracyTest(20, context)
	// mulParfullyConnectedAccuracyTest(20, context) //conventional
	// rotOptDownSamplingTest(context)
	// mulParDownSamplingTest(context)
	// reluTest(context)

	// RotKey Test
	// mulParConvRotKeyTest(20)
	// HierarchyKeyTest(20)
	// generalKeyTest(context)

	// Print Blue Print
	// getBluePrint()

	//Resnet Inference Tests
	// resnetInferenceTest(20, context) //You have to enable myLogsSave codes in resnet.go to use this function.
	// resnetprevInferenceTest(20, context)
	// resnetInferenceForCifar10(20, context)

	//basicOperationTimeTest
	// basicOperationTimeTest(context)

	/////////////////////////////////////
	//Matrix-Vector Multiplication Test//
	/////////////////////////////////////
	parBsgsMatVecMultAccuracyTest(context) //proposed
	bsgsMatVecMultAccuracyTest(context)    //conventional
}
