package main

import (
	"fmt"
	"os"
	"rotopt/engine"
	"sort"
	"time"
	"unsafe"

	"github.com/tuneinsight/lattigo/v5/core/rlwe"
	"github.com/tuneinsight/lattigo/v5/schemes/ckks"
)

func main() {

	args := os.Args[1:]
	//example :
	//go run . conv parBSGS rotkey

	//== supported args ======================================================================================================================================//
	// basic : Execution time of rotation, multiplication, addition in our CKKS environment.
	// conv : Time comparison between rotation optimized convolution and multiplexed parallel convolution.
	// blueprint : Extract current blueprint.
	// downsamp : Time comparison between rotation optimized downsampling and multiplexed parllel downsampling.
	// rotkey : Rotation key reduction test.
	// parBSGS :Time comparison between Parallel BSGS matrix vector multiplication and BSGS diagonal method.
	// fc : Apply Parallel BSGS matrix vector multiplication to Fully Connected Layer of Resnet20(CIFAR-10 images.) where matrx(10x64) vector(64x1) result(10x1).
	//=========================================================================================================================================================//

	if len(args) == 0 {
		fmt.Println("args set as ALL")
		args = []string{"ALL"}
	} else if len(args) == 1 {
		fmt.Println("args : ", args)
	}

	//CKKS settings
	context := setCKKSEnv() //CKKS environment

	//basicOperationTimeTest
	if Contains(args, "basic") || args[0] == "ALL" {
		fmt.Println("\nBasic operation time test started!")
		basicOperationTimeTest(context)
	}

	///////////////////////////////////////
	//Rotation Optimized Convolution Test//
	///////////////////////////////////////

	// Convolution Tests
	if Contains(args, "conv") || args[0] == "ALL" {
		rotOptConvTimeTest(context, 2)
		rotOptConvTimeTest(context, 3)
		rotOptConvTimeTest(context, 4)
		rotOptConvTimeTest(context, 5)
		mulParConvTimeTest(context)
	}

	// Print Blue Print. Corresponds to Appendix A.
	if Contains(args, "blueprint") || args[0] == "ALL" {
		getBluePrint()
	}

	// Downsampling Tests
	if Contains(args, "downsamp") || args[0] == "ALL" {
		rotOptDownSamplingTest(context)
		mulParDownSamplingTest(context)
	}

	// RotKey Test
	if Contains(args, "rotkey") || args[0] == "ALL" {
		HierarchyKeyTest()      //Hierarchical two-level rotation key system.
		overallKeyTest(context) //Also apply small level key system.
	}

	/////////////////////////////////////
	//Matrix-Vector Multiplication Test//
	/////////////////////////////////////
	//Apply to fully connected layer
	if Contains(args, "fc") || args[0] == "ALL" {
		parBSGSfullyConnectedAccuracyTest(context) //using parallel BSGS matrix-vector multiplication to fully connected layer.
		mulParfullyConnectedAccuracyTest(context)  //conventional
	}
	if Contains(args, "parBSGS") || args[0] == "ALL" {
		for N := 32; N <= 512; N *= 2 {
			parBsgsMatVecMultAccuracyTest(N, context) //proposed
			bsgsMatVecMultAccuracyTest(N, context)    //conventional
		}
	}

	/////////////////////////////////////
	///////////////Revision//////////////
	/////////////////////////////////////
	//Accuracy, Recall, F1 score
	if Contains(args, "parBSGS") || args[0] == "ALL" {
		for N := 32; N <= 512; N *= 2 {
			parBsgsMatVecMultAccuracyTest(N, context) //proposed
			bsgsMatVecMultAccuracyTest(N, context)    //conventional
		}
	}
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

func rotOptConvTimeTest(cc *customContext, depth int) {
	fmt.Printf("\nRotation Optimized Convolution (for %d-depth consumed) time test started!\n", depth)

	var convIDs []string
	var maxDepth []int

	switch depth {
	case 2:
		convIDs = []string{"CONV1", "CONV2", "CONV3s2", "CONV3", "CONV4s2", "CONV4"}
		maxDepth = []int{2, 2, 2, 2, 2, 2}
	case 3:
		convIDs = []string{"CONV2", "CONV3s2", "CONV3", "CONV4s2", "CONV4"}
		maxDepth = []int{3, 3, 3, 3, 3}
	case 4:
		convIDs = []string{"CONV2", "CONV3s2", "CONV3", "CONV4s2", "CONV4"}
		maxDepth = []int{4, 4, 4, 4, 4}
	case 5:
		convIDs = []string{"CONV3s2", "CONV4s2"}
		maxDepth = []int{5, 5}
	default:
		fmt.Printf("Unsupported depth: %d\n", depth)
		return
	}

	iter := 1
	minStartCipherLevel := depth
	maxStartCipherLevel := cc.Params.MaxLevel()

	for index := 0; index < len(convIDs); index++ {
		for d := depth; d <= maxDepth[index]; d++ {
			convID := convIDs[index]

			//register index of rotation
			rots := engine.RotOptConvRegister(convID, d)

			//rotation key register
			newEvaluator := rotIndexToGaloisEl(int2dTo1d(rots), cc.Params, cc.Kgen, cc.Sk)

			//make rotOptConv instance
			conv := engine.NewrotOptConv(newEvaluator, cc.Encoder, cc.Params, 20, convID, d, getConvTestNum(convID)[0], getConvTestNum(convID)[1])

			// Make input and kernel
			cf := conv.ConvFeature
			plainInput := makeRandomInput(cf.InputDataChannel, cf.InputDataHeight, cf.InputDataWidth)
			plainKernel := makeRandomKernel(cf.KernelNumber, cf.InputDataChannel, cf.KernelSize, cf.KernelSize)

			//Plaintext Convolution
			plainOutput := PlainConvolution2D(plainInput, plainKernel, cf.Stride, 1)

			// Encrypt Input, Encode Kernel
			mulParPackedInput := MulParPacking(plainInput, cf, cc)
			conv.PreCompKernels = EncodeKernel(plainKernel, cf, cc)

			fmt.Printf("=== convID : %s, Depth : %v, CipherLevel : %v ~ %v, iter : %v === \n", convID, d, Max(minStartCipherLevel, d), maxStartCipherLevel, iter)
			fmt.Printf("startLevel executionTime(sec)\n")
			// MSE, RE, inf Norm
			var MSEList, REList, infNormList []float64
			for startCipherLevel := Max(minStartCipherLevel, d); startCipherLevel <= maxStartCipherLevel; startCipherLevel++ {
				plain := ckks.NewPlaintext(cc.Params, startCipherLevel)
				cc.Encoder.Encode(mulParPackedInput, plain)
				inputCt, _ := cc.EncryptorSk.EncryptNew(plain)

				var totalForwardTime time.Duration
				//Conv Foward
				for i := 0; i < iter; i++ {
					//Convolution start
					start := time.Now()
					encryptedOutput := conv.Foward(inputCt)
					end := time.Now()
					totalForwardTime += end.Sub(start)

					//Acc, Recall, F1 score
					FHEOutput := UnMulParPacking(encryptedOutput, cf, cc)
					scores := MSE_RE_infNorm(plainOutput, FHEOutput)
					MSEList = append(MSEList, scores[0])
					REList = append(REList, scores[1])
					infNormList = append(infNormList, scores[2])
				}

				//Print Elapsed Time
				avgForwardTime := float64(totalForwardTime.Nanoseconds()) / float64(iter) / 1e9
				fmt.Printf("%v %v \n", startCipherLevel, avgForwardTime)
			}
			// Average Acc, Recall, F1 score
			MSEMin, MSEMax, MSEAvg := minMaxAvg(MSEList)
			REMin, REMax, REAvg := minMaxAvg(REList)
			infNormMin, infNormMax, infNormAvg := minMaxAvg(infNormList)

			fmt.Printf("MSE (Mean Squared Error)   : Min = %.2e, Max = %.2e, Avg = %.2e\n", MSEMin, MSEMax, MSEAvg)
			fmt.Printf("Relative Error             : Min = %.2e, Max = %.2e, Avg = %.2e\n", REMin, REMax, REAvg)
			fmt.Printf("Infinity Norm (L-infinity) : Min = %.2e, Max = %.2e, Avg = %.2e\n", infNormMin, infNormMax, infNormAvg)
			fmt.Println()
		}
	}
}
func mulParConvTimeTest(cc *customContext) {
	fmt.Println("\nMultiplexed Parallel Convolution time test started!")

	convIDs := []string{"CONV1", "CONV2", "CONV3s2", "CONV3", "CONV4s2", "CONV4"}

	//Set iter
	iter := 1

	minStartCipherLevel := 2
	maxStartCipherLevel := cc.Params.MaxLevel()

	for index := 0; index < len(convIDs); index++ {
		convID := convIDs[index]

		//register index of rotation
		rots := engine.MulParConvRegister(convID)

		//rotation key register
		newEvaluator := rotIndexToGaloisEl(int2dTo1d(rots), cc.Params, cc.Kgen, cc.Sk)

		//make mulParConv instance
		conv := engine.NewMulParConv(newEvaluator, cc.Encoder, cc.Params, 20, convID, getConvTestNum(convID)[0], getConvTestNum(convID)[1])

		// Make input and kernel
		cf := conv.ConvFeature
		plainInput := makeRandomInput(cf.InputDataChannel, cf.InputDataHeight, cf.InputDataWidth)
		plainKernel := makeRandomKernel(cf.KernelNumber, cf.InputDataChannel, cf.KernelSize, cf.KernelSize)

		//Plaintext Convolution
		plainOutput := PlainConvolution2D(plainInput, plainKernel, cf.Stride, 1)

		// Encrypt Input, Encode Kernel
		mulParPackedInput := MulParPacking(plainInput, cf, cc)
		conv.PreCompKernels = EncodeKernel(plainKernel, cf, cc)

		// Multiplexed parallel convolution Start!
		fmt.Printf("=== convID : %s, Depth : %v, CipherLevel : %v ~ %v, iter : %v === \n", convID, 2, minStartCipherLevel, maxStartCipherLevel, iter)
		fmt.Printf("startLevel executionTime(sec)\n")
		// MSE, RE, inf Norm
		var MSEList, REList, infNormList []float64
		for startCipherLevel := minStartCipherLevel; startCipherLevel <= maxStartCipherLevel; startCipherLevel++ {

			plain := ckks.NewPlaintext(cc.Params, startCipherLevel)
			cc.Encoder.Encode(mulParPackedInput, plain)
			inputCt, _ := cc.EncryptorSk.EncryptNew(plain)

			var totalForwardTime time.Duration
			//Conv Foward
			for i := 0; i < iter; i++ {
				//Convolution start
				start := time.Now()
				encryptedOutput := conv.Foward(inputCt)
				end := time.Now()
				totalForwardTime += end.Sub(start)

				//Acc, Recall, F1 score
				FHEOutput := UnMulParPacking(encryptedOutput, cf, cc)
				scores := MSE_RE_infNorm(plainOutput, FHEOutput)
				MSEList = append(MSEList, scores[0])
				REList = append(REList, scores[1])
				infNormList = append(infNormList, scores[2])
			}

			//Print Elapsed Time
			avgForwardTime := float64(totalForwardTime.Nanoseconds()) / float64(iter) / 1e9
			fmt.Printf("%v %v \n", startCipherLevel, avgForwardTime)
		}
		// Average Acc, Recall, F1 score
		MSEMin, MSEMax, MSEAvg := minMaxAvg(MSEList)
		REMin, REMax, REAvg := minMaxAvg(REList)
		infNormMin, infNormMax, infNormAvg := minMaxAvg(infNormList)

		fmt.Printf("MSE (Mean Squared Error)   : Min = %.2e, Max = %.2e, Avg = %.2e\n", MSEMin, MSEMax, MSEAvg)
		fmt.Printf("Relative Error             : Min = %.2e, Max = %.2e, Avg = %.2e\n", REMin, REMax, REAvg)
		fmt.Printf("Infinity Norm (L-infinity) : Min = %.2e, Max = %.2e, Avg = %.2e\n", infNormMin, infNormMax, infNormAvg)
		fmt.Println()
	}
}
func parBSGSfullyConnectedAccuracyTest(cc *customContext) {
	fmt.Println("Fully Connected + Parallel BSGS matrix-vector multiplication Test!")
	startLevel := 1
	endLevel := cc.Params.MaxLevel()
	// endLevel := 2
	//register
	rot := engine.ParBSGSFCRegister()

	//rot register
	newEvaluator := RotIndexToGaloisElements(rot, cc)

	//make avgPooling instance
	fc := engine.NewParBSGSFC(newEvaluator, cc.Encoder, cc.Params, 20)

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
	fmt.Printf("startLevel executionTime\n")
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
		// fmt.Printf("%v %v \n", level, TimeDurToFloatSec(endTime.Sub(startTime)))
		fmt.Printf("%v %v \n", level, (endTime.Sub(startTime)))

	}

	//Decryption
	outputFloat := ciphertextToFloat(outputCt, cc)

	fmt.Println("Accuracy : ", euclideanDistance(outputFloat[0:10], trueOutputFloat))
}

func mulParfullyConnectedAccuracyTest(cc *customContext) {
	fmt.Println("Fully Connected + Conventional BSGS diagonal matrix-vector multiplication Test!")
	startLevel := 1
	endLevel := cc.Params.MaxLevel()
	//register
	rot := engine.MulParFCRegister()

	//rot register
	newEvaluator := RotIndexToGaloisElements(rot, cc)

	//make avgPooling instance
	fc := engine.NewMulParFC(newEvaluator, cc.Encoder, cc.Params, 20)

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
	fmt.Printf("startLevel executionTime\n")
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
		// fmt.Printf("%v %v \n", level, TimeDurToFloatSec(endTime.Sub(startTime)))
		fmt.Printf("%v %v \n", level, (endTime.Sub(startTime)))

	}

	//Decryption
	outputFloat := ciphertextToFloat(outputCt, cc)

	fmt.Println("Accuracy : ", euclideanDistance(outputFloat[0:10], trueOutputFloat))

}

func rotOptDownSamplingTest(cc *customContext) {
	fmt.Println("Rotation Optimized Downsampling Test started! ")
	//register
	rot := engine.RotOptDSRegister()

	//rot register
	newEvaluator := RotIndexToGaloisElements(rot, cc)

	//make avgPooling instance
	ds16 := engine.NewRotOptDS(16, newEvaluator, cc.Encoder, cc.Params)
	ds32 := engine.NewRotOptDS(32, newEvaluator, cc.Encoder, cc.Params)

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
		// fmt.Printf("%v Time(DONWSAMP1) : %v \n", level, TimeDurToFloatSec(endTime.Sub(startTime)))
		fmt.Printf("%v Time(DONWSAMP1) : %v \n", level, (endTime.Sub(startTime)))

		// ////////
		// Timer start
		startTime = time.Now()

		// AvgPooling Foward
		ds32.Foward(inputCt)

		// Timer end
		endTime = time.Now()

		// Print Elapsed Time
		// fmt.Printf("%v Time(DOWNSAMP2) : %v \n", level, TimeDurToFloatSec(endTime.Sub(startTime)))
		fmt.Printf("%v Time(DOWNSAMP2) : %v \n", level, (endTime.Sub(startTime)))

	}

}

func mulParDownSamplingTest(cc *customContext) {
	fmt.Println("Multiplexed Parallel Downsampling Test started! ")
	//register
	rot := engine.MulParDSRegister()

	//rot register
	newEvaluator := RotIndexToGaloisElements(rot, cc)

	//make avgPooling instance
	ds16 := engine.NewMulParDS(16, newEvaluator, cc.Encoder, cc.Params)
	ds32 := engine.NewMulParDS(32, newEvaluator, cc.Encoder, cc.Params)

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
		// fmt.Printf("%v Time(16) : %v \n", level, TimeDurToFloatSec(endTime.Sub(startTime)))
		fmt.Printf("%v Time(DOWNSAMP1) : %v \n", level, (endTime.Sub(startTime)))

		// ////////
		// Timer start
		startTime = time.Now()

		// AvgPooling Foward
		ds32.Foward(inputCt)

		// Timer end
		endTime = time.Now()

		// Print Elapsed Time
		// fmt.Printf("%v Time(32) : %v \n", level, TimeDurToFloatSec(endTime.Sub(startTime)))
		fmt.Printf("%v Time(DOWNSAMP2) : %v \n", level, (endTime.Sub(startTime)))

	}

	// outputFloat16 := ciphertextToFloat(outputCt16, cc)
	// outputFloat32 := ciphertextToFloat(outputCt32, cc)
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
		LogQ: []int{51,
			46, 46, 46, 46, 46,
			46, 46, 46, 46, 46,
			46, 46, 46, 46, 46,
			46, 46, 46, 46, 46,
			46, 46, 46, 46},
		LogP:            []int{60, 60, 60, 60, 60},
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

	return context
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
func HierarchyKeyTest() {
	fmt.Println("Hierarchical Key system applied Test!")
	//Organize what kinds of key-level 0 keys needed.
	mulParRot, rotOptRot := RotKeyOrganize(20)

	//For rotOptRot
	fmt.Println("===Rotation Optimized Convolution Level1 key needed===")
	fmt.Println("T : ", Level1RotKeyNeededForInference(rotOptRot))
	//For MulPar
	fmt.Println("====Multiplexed Parallel Convolution Level1 key needed===")
	fmt.Println("T : ", Level1RotKeyNeededForInference(mulParRot))

}

func overallKeyTest(cc *customContext) {
	fmt.Println("Hierarchical Key system, Small level key system applied Test!")

	hdnum := 4.0

	//register
	convIDs := []string{"CONV1", "CONV2", "CONV3s2", "CONV3", "CONV4s2", "CONV4"}
	maxDepth := []int{2, 2, 2, 2, 2, 2}

	mulPar := make([][]int, 3)
	rotOpt := make([][]int, 3)
	for index := 0; index < len(convIDs); index++ {
		mulParRot := engine.MulParConvRegister(convIDs[index])
		rotOptRot := engine.RotOptConvRegister(convIDs[index], maxDepth[index])

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

	// fmt.Println(mulPar)
	// fmt.Println(rotOpt)

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

	fmt.Println("Warning! Key size might be different due to some lattigo automatic parameter setting")
	// With max Mult Level , 0 key level
	multMaxkey0 := NewSmallLevelKey(1, 0, cc.Params.MaxLevel(), &cc.Params)
	eachKeySize := multMaxkey0.GetKeySize()
	fmt.Println("==MulPar with max Mult Level, 0 key level==")
	fmt.Println(float64(eachKeySize*len(linmulPar))/1048576.0, "MB")
	fmt.Println("==RotOpt with max Mult Level, 0 key level==")
	fmt.Println(float64(eachKeySize*len(linrotOpt))/1048576.0, "MB")

	// With max Mult Level , 1 key level
	lv1keys := []int{1, -1, 4, -4, 16, -16, 256, -256, 1024, -1024, 4096, -4096, 16384, -16384}
	multMaxkey1 := GenLevelUpKey(multMaxkey0, hdnum) //multMaxkey0.Hdnum
	eachKeySize = multMaxkey1.GetKeySize()

	// fmt.Println("MulPar with max Mult Level, 1 key level")
	// fmt.Println(eachKeySize*len(lv1keys)/1048576, "MB")
	fmt.Println("==RotOpt with max Mult Level, 1 key level==")
	fmt.Println(float64(eachKeySize*len(lv1keys))/1048576.0, "MB")

	// With opt Mult Level , 1 key level
	// fmt.Println("MulPar with opt Mult Level, 1 key level")
	fmt.Println("==RotOpt with opt Mult Level, 0 key level==")
	mult2key0 := NewSmallLevelKey(1, 0, 2, &cc.Params)
	eachKeySize = mult2key0.GetKeySize()
	fmt.Println(float64(eachKeySize*len(linrotOpt))/1048576.0, "MB")

	//Final. Opt Mult Level, 1 key level
	fmt.Println("==RotOpt with opt Mult Level, 1 key level==")
	mult2key1 := GenLevelUpKey(mult2key0, hdnum)
	eachKeySize = mult2key1.GetKeySize()
	fmt.Println(float64(eachKeySize*len(linrotOpt))/1048576.0, "MB")

}

// Extract current blueprint
func getBluePrint() {
	fmt.Println("Blue Print test started! Display all blueprint for convolution optimized convolutions.")
	fmt.Println("You can test other blue prints in engine/convConfig.go")

	convIDs := []string{"CONV1", "CONV2", "CONV3s2", "CONV3", "CONV4s2", "CONV4"}
	maxDepth := []int{2, 4, 5, 4, 5, 4}

	for index := 0; index < len(convIDs); index++ {
		for depth := 2; depth <= maxDepth[index]; depth++ {
			fmt.Printf("=== convID : %s, depth : %v === \n", convIDs[index], depth)
			convMap, _, _ := engine.GetConvBlueprints(convIDs[index], depth)
			rotSumBP := make([][]int, 1)
			rotSumBP[0] = []int{0}
			crossCombineBP := make([]int, 0)

			for d := 1; d < len(convMap); d++ {

				if convMap[d][0] == 3 {
					crossCombineBP = append(crossCombineBP, convMap[d][1])
					crossCombineBP = append(crossCombineBP, 0)
					crossCombineBP = append(crossCombineBP, convMap[d][2:]...)
					break
				} else {
					rotSumBP = append(rotSumBP, convMap[d])
				}

			}
			rotSumBP[0][0] = len(rotSumBP) - 1

			fmt.Println("RotationSumBP : ")
			fmt.Print("[")
			for _, row := range rotSumBP {
				fmt.Print("[")
				for i, val := range row {
					if i > 0 {
						fmt.Print(", ")
					}
					fmt.Printf("%d", val)
				}
				fmt.Print("],")
			}
			fmt.Println("]")

			fmt.Println("CrossCombineBP : ")
			fmt.Print("[")
			for i, val := range crossCombineBP {
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Printf("%d", val)
			}
			fmt.Println("]")

			fmt.Println("KernelBP : ")
			fmt.Print("[")
			for _, row := range engine.GetMulParConvFeature(convIDs[index]).KernelBP {
				fmt.Print("[")
				for i, val := range row {
					if i > 0 {
						fmt.Print(", ")
					}
					fmt.Printf("%d", val)
				}
				fmt.Print("],")
			}
			fmt.Println("]")
			fmt.Println()

		}

	}

}

func basicOperationTimeTest(cc *customContext) {
	floats := makeRandomFloat(32768)

	rot := make([]int, 1)
	rot[0] = 1

	fmt.Println("StartLevel Rotate Add Mul")
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

		start3 := time.Now()
		newEvaluator.Mul(cipher1, cipher1, cipher1)
		end3 := time.Now()
		// newEvaluator.Rescale(cipher1, cipher1)
		// fmt.Println(i, TimeDurToFloatMiliSec(end1.Sub(start1)), TimeDurToFloatMiliSec(end2.Sub(start2)), TimeDurToFloatMiliSec(end3.Sub(start3)))
		fmt.Println(i, end1.Sub(start1), end2.Sub(start2), end3.Sub(start3))
	}

}

func bsgsMatVecMultAccuracyTest(N int, cc *customContext) {
	fmt.Println("Conventional BSGS diagonal matrix-vector multiplication Test!")
	fmt.Println("matrix : ", N, "x", N, "  vector : ", N, "x", 1)
	nt := 32768

	fmt.Printf("=== Conevntional (BSGS diag mat(N*N)-vec(N*1) mul) method start! N : %v ===\n", N)

	A := getPrettyMatrix(N, N)
	B := getPrettyMatrix(N, 1)

	//answer
	answer := originalMatMul(A, B)

	//change B to ciphertext
	B1d := make2dTo1d(B)
	B1d = resize(B1d, nt)
	//start mat vec mul
	rot := engine.BsgsDiagMatVecMulRegister(N)
	newEvaluator := RotIndexToGaloisElements(rot, cc)
	matVecMul := engine.NewBsgsDiagMatVecMul(A, N, nt, newEvaluator, cc.Encoder, cc.Params)

	fmt.Printf("startLevel executionTime\n")
	for level := 1; level <= cc.Params.MaxLevel(); level++ {

		Bct := floatToCiphertextLevel(B1d, level, cc.Params, cc.Encoder, cc.EncryptorSk)

		startTime := time.Now()
		BctOut := matVecMul.Foward(Bct)
		endTime := time.Now()
		outputFloat := ciphertextToFloat(BctOut, cc)

		euclideanDistance(outputFloat[0:N], make2dTo1d(answer))
		// fmt.Println(level, TimeDurToFloatSec(endTime.Sub(startTime)))
		fmt.Println(level, endTime.Sub(startTime))
	}
}
func parBsgsMatVecMultAccuracyTest(N int, cc *customContext) {
	fmt.Println("Parallel BSGS matrix-vector multiplication Test!")
	fmt.Println("matrix : ", N, "x", N, "  vector : ", N, "x", 1)
	nt := cc.Params.MaxSlots()
	pi := 1 //initially setting. (how many identical datas are in single ciphertext)

	fmt.Printf("=== Proposed (Parallely BSGS diag mat(N*N)-vec(N*1) mul) method start! N : %v ===\n", N)

	A := getPrettyMatrix(N, N)
	B := getPrettyMatrix(N, 1)

	answer := originalMatMul(A, B)

	B1d := make2dTo1d(B)
	B1d = resize(B1d, nt)
	for i := 1; i < pi; i *= 2 {
		tempB := rotate(B1d, -(nt/pi)*i)
		B1d = add(tempB, B1d)
	}
	//start mat vec mul
	rot := engine.ParBsgsDiagMatVecMulRegister(N, nt, pi)
	newEvaluator := RotIndexToGaloisElements(rot, cc)
	matVecMul := engine.NewParBsgsDiagMatVecMul(A, N, nt, pi, newEvaluator, cc.Encoder, cc.Params)

	fmt.Printf("startLevel executionTime\n")
	for level := 1; level <= cc.Params.MaxLevel(); level++ {

		Bct := floatToCiphertextLevel(B1d, level, cc.Params, cc.Encoder, cc.EncryptorSk)

		startTime := time.Now()
		BctOut := matVecMul.Foward(Bct)
		endTime := time.Now()
		outputFloat := ciphertextToFloat(BctOut, cc)

		euclideanDistance(outputFloat[0:N], make2dTo1d(answer))
		// fmt.Println(level, TimeDurToFloatSec(endTime.Sub(startTime)))
		fmt.Println(level, endTime.Sub(startTime))
	}
}
func Contains(slice []string, str string) bool {
	for _, v := range slice {
		if v == str {
			return true
		}
	}
	return false
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
func MakeGalois(cc *customContext, rotIndexes [][]int) [][]*rlwe.GaloisKey {

	galEls := make([][]*rlwe.GaloisKey, len(rotIndexes))

	for level := 0; level < len(rotIndexes); level++ {
		var galElements []uint64
		for _, rot := range rotIndexes[level] {
			galElements = append(galElements, cc.Params.GaloisElement(rot))
		}
		galKeys := cc.Kgen.GenGaloisKeysNew(galElements, cc.Sk)

		galEls = append(galEls, galKeys)

		fmt.Println(unsafe.Sizeof(*galKeys[0]), unsafe.Sizeof(galKeys[0].GaloisElement), unsafe.Sizeof(galKeys[0].NthRoot), unsafe.Sizeof(galKeys[0].EvaluationKey), unsafe.Sizeof(galKeys[0].GadgetCiphertext), unsafe.Sizeof(galKeys[0].BaseTwoDecomposition), unsafe.Sizeof(galKeys[0].Value))
	}
	// newEvaluator := ckks.NewEvaluator(cc.Params, rlwe.NewMemEvaluationKeySet(cc.Kgen.GenRelinearizationKeyNew(cc.Sk), galKeys...))
	return galEls
}
