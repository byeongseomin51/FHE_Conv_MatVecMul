package main

import (
	"fmt"
	"os"
	"rotopt/modules"
	"sort"
	"time"
	"unsafe"

	"github.com/tuneinsight/lattigo/v5/core/rlwe"
	"github.com/tuneinsight/lattigo/v5/ring"
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
	// context := setCKKSEnv() //default CKKS environment
	context := setCKKSEnvUseParamSet("PN15QP880CI") //Lightest parameter set

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
	if Contains(args, "matVecMul") || args[0] == "ALL" {
		for N := 32; N <= 512; N *= 2 {
			parBsgsMatVecMultAccuracyTest(N, context) //proposed
			bsgsMatVecMultAccuracyTest(N, context)    //conventional
		}
	}

	/////////////////////////////////////
	///////////////Revision//////////////
	/////////////////////////////////////
	//args["conv"] updated. (accuracy calculate)

	// CKKS parameter settings
	if Contains(args, "paramTest") || args[0] == "ALL" {
		CKKSEnvSetList := []string{"PN16QP1761", "PN15QP880CI", "PN16QP1654pq", "PN15QP827CIpq"}
		for _, SetName := range CKKSEnvSetList {
			contextCustom := setCKKSEnvUseParamSet(SetName)

			rotOptConvTimeTest(contextCustom, 2)
			rotOptConvTimeTest(contextCustom, 3)
			rotOptConvTimeTest(contextCustom, 4)
			rotOptConvTimeTest(contextCustom, 5)
			mulParConvTimeTest(contextCustom)

			parBSGSfullyConnectedAccuracyTest(context) //using parallel BSGS matrix-vector multiplication to fully connected layer.
			mulParfullyConnectedAccuracyTest(context)  //conventional
			for N := 32; N <= 512; N *= 2 {
				parBsgsMatVecMultAccuracyTest(N, context) //proposed
				bsgsMatVecMultAccuracyTest(N, context)    //conventional
			}
		}
	}

	// Generalization of different AI models
	if Contains(args, "otherConv") || args[0] == "ALL" {
		// Each convolution refers to...
		// CvTCifar100Stage2, CvTCifar100Stage3 : convolutional embedding in CvT (Convolutional Vision Transformer) model.
		// MUSE_PyramidGenConv 			  		: create a multi-scale feature pyramid from a single-scale feature map in MUSE (a model based on Mamba). https://ojs.aaai.org/index.php/AAAI/article/view/32778
		otherMulParConvTimeTest(context)
		otherRotOptConvTimeTest(context, 2)
		otherRotOptConvTimeTest(context, 3)
		otherRotOptConvTimeTest(context, 4)
		otherRotOptConvTimeTest(context, 5)
		otherRotOptConvTimeTest(context, 6)
		otherRotOptConvTimeTest(context, 7)
		otherRotOptConvTimeTest(context, 8)
		otherRotOptConvTimeTest(context, 9)
		otherRotOptConvTimeTest(context, 10)
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

func otherRotOptConvTimeTest(cc *customContext, depth int) {
	fmt.Printf("\nRotation Optimized Convolution (for %d-depth consumed, complex AI model) time test started!\n", depth)

	var convIDs []string

	switch depth {
	case 2:
		convIDs = []string{"CvTCifar100Stage2", "CvTCifar100Stage3", "MUSE_PyramidGenConv"}
	case 3:
		convIDs = []string{"CvTCifar100Stage2", "CvTCifar100Stage3", "MUSE_PyramidGenConv"}
	case 4:
		convIDs = []string{"CvTCifar100Stage2", "CvTCifar100Stage3", "MUSE_PyramidGenConv"}
	case 5:
		convIDs = []string{"CvTCifar100Stage2", "CvTCifar100Stage3", "MUSE_PyramidGenConv"}
	case 6:
		convIDs = []string{"CvTCifar100Stage3", "MUSE_PyramidGenConv"}
	case 7:
		convIDs = []string{"MUSE_PyramidGenConv"}
	case 8:
		convIDs = []string{"MUSE_PyramidGenConv"}
	case 9:
		convIDs = []string{"MUSE_PyramidGenConv"}
	case 10:
		convIDs = []string{"MUSE_PyramidGenConv"}
	default:
		fmt.Printf("Unsupported depth: %d\n", depth)
		return
	}

	iter := 1
	minStartCipherLevel := depth
	maxStartCipherLevel := cc.Params.MaxLevel() //원
	// maxStartCipherLevel := depth

	for index := 0; index < len(convIDs); index++ {

		convID := convIDs[index]
		if convID == "MUSE_PyramidGenConv" {
			cc = setCKKSEnvUseParamSet("PN15QP880CI")

			maxStartCipherLevel = cc.Params.MaxLevel()
		}

		//register index of rotation
		rots := modules.RotOptConvRegister(convID, depth)
		// fmt.Println(len(rots[0]), len(rots[1]), len(rots[2]), len(rots[0])+len(rots[1])+len(rots[2]))
		// continue

		//rotation key register
		newEvaluator := rotIndexToGaloisEl(int2dTo1d(rots), cc.Params, cc.Kgen, cc.Sk)

		//make rotOptConv instance
		conv := modules.NewrotOptConv(newEvaluator, cc.Encoder, cc.Params, convID, depth)

		// Make input and kernel
		cf := conv.ConvFeature
		plainInput := makeRandomInput(cf.InputDataChannel, cf.InputDataHeight, cf.InputDataWidth)
		plainKernel := makeRandomKernel(cf.KernelNumber, cf.InputDataChannel, cf.KernelSize, cf.KernelSize)

		//Plaintext Convolution
		plainOutput := PlainConvolution2D(plainInput, plainKernel, cf.Stride, 1)

		// Encrypt Input, Encode Kernel
		mulParPackedInput := MulParPacking(plainInput, cf, cc)
		conv.PreCompKernels = EncodeKernel(plainKernel, cf, cc)

		fmt.Printf("=== convID : %s, Depth : %v, CipherLevel : %v ~ %v, iter : %v === \n", convID, depth, Max(minStartCipherLevel, depth), maxStartCipherLevel, iter)
		fmt.Printf("startLevel executionTime(sec)\n")
		// MSE, RE, inf Norm
		var MSEList, REList, infNormList []float64
		for startCipherLevel := Max(minStartCipherLevel, depth); startCipherLevel <= maxStartCipherLevel; startCipherLevel++ {
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

				// MSE, RE, inf Norm
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
func otherMulParConvTimeTest(cc *customContext) {
	fmt.Println("\nMultiplexed Parallel Convolution (for complex AI model) time test started!")

	convIDs := []string{"CvTCifar100Stage2", "CvTCifar100Stage3", "MUSE_PyramidGenConv"}
	// convIDs := []string{"MUSE_PyramidGenConv"}
	//Set iter
	iter := 1

	minStartCipherLevel := 2
	maxStartCipherLevel := cc.Params.MaxLevel()
	// maxStartCipherLevel := 2

	for index := 0; index < len(convIDs); index++ {
		// Get ConvID
		convID := convIDs[index]
		if convID == "MUSE_PyramidGenConv" {
			cc = setCKKSEnvUseParamSet("PN15QP880CI")
			fmt.Println("CKKS parameter set as : PN15QP880CI")

			maxStartCipherLevel = cc.Params.MaxLevel()
		}

		//register index of rotation
		rots := modules.MulParConvRegister(convID)
		// fmt.Println(len(rots[0]), len(rots[1]), len(rots[2]), len(rots[0])+len(rots[1])+len(rots[2]))
		// continue

		//rotation key register
		newEvaluator := rotIndexToGaloisEl(int2dTo1d(rots), cc.Params, cc.Kgen, cc.Sk)

		//make mulParConv instance
		conv := modules.NewMulParConv(newEvaluator, cc.Encoder, cc.Params, convID)

		// Make input and kernel
		cf := conv.ConvFeature
		plainInput := makeRandomInput(cf.InputDataChannel, cf.InputDataHeight, cf.InputDataWidth)
		plainKernel := makeRandomKernel(cf.KernelNumber, cf.InputDataChannel, cf.KernelSize, cf.KernelSize)

		//Plaintext Convolution
		plainOutput := PlainConvolution2D(plainInput, plainKernel, cf.Stride, 1)
		// print3DArray(plainOutput)

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

				// MSE, RE, inf Norm
				FHEOutput := UnMulParPacking(encryptedOutput, cf, cc)
				// print3DArray(FHEOutput)
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
func rotOptConvTimeTest(cc *customContext, depth int) {
	fmt.Printf("\nRotation Optimized Convolution (for %d-depth consumed) time test started!\n", depth)

	var convIDs []string

	switch depth {
	case 2:
		convIDs = []string{"CONV1", "CONV2", "CONV3s2", "CONV3", "CONV4s2", "CONV4"}
	case 3:
		convIDs = []string{"CONV2", "CONV3s2", "CONV3", "CONV4s2", "CONV4"}
	case 4:
		convIDs = []string{"CONV2", "CONV3s2", "CONV3", "CONV4s2", "CONV4"}
	case 5:
		convIDs = []string{"CONV3s2", "CONV4s2"}
	default:
		fmt.Printf("Unsupported depth: %d\n", depth)
		return
	}

	iter := 1
	minStartCipherLevel := depth
	maxStartCipherLevel := cc.Params.MaxLevel()
	// maxStartCipherLevel := depth

	for index := 0; index < len(convIDs); index++ {

		convID := convIDs[index]

		//register index of rotation
		rots := modules.RotOptConvRegister(convID, depth)

		//rotation key register
		newEvaluator := rotIndexToGaloisEl(int2dTo1d(rots), cc.Params, cc.Kgen, cc.Sk)

		//make rotOptConv instance
		conv := modules.NewrotOptConv(newEvaluator, cc.Encoder, cc.Params, convID, depth)

		// Make input and kernel
		cf := conv.ConvFeature
		plainInput := makeRandomInput(cf.InputDataChannel, cf.InputDataHeight, cf.InputDataWidth)
		plainKernel := makeRandomKernel(cf.KernelNumber, cf.InputDataChannel, cf.KernelSize, cf.KernelSize)

		//Plaintext Convolution
		plainOutput := PlainConvolution2D(plainInput, plainKernel, cf.Stride, 1)

		// Encrypt Input, Encode Kernel
		mulParPackedInput := MulParPacking(plainInput, cf, cc)
		conv.PreCompKernels = EncodeKernel(plainKernel, cf, cc)

		fmt.Printf("=== convID : %s, Depth : %v, CipherLevel : %v ~ %v, iter : %v === \n", convID, depth, Max(minStartCipherLevel, depth), maxStartCipherLevel, iter)
		fmt.Printf("startLevel executionTime(sec)\n")
		// MSE, RE, inf Norm
		var MSEList, REList, infNormList []float64
		for startCipherLevel := Max(minStartCipherLevel, depth); startCipherLevel <= maxStartCipherLevel; startCipherLevel++ {
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

				// MSE, RE, inf Norm
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
func mulParConvTimeTest(cc *customContext) {
	fmt.Println("\nMultiplexed Parallel Convolution time test started!")

	convIDs := []string{"CONV1", "CONV2", "CONV3s2", "CONV3", "CONV4s2", "CONV4"}
	// convIDs := []string{"CONV4"}

	//Set iter
	iter := 1

	minStartCipherLevel := 2
	maxStartCipherLevel := cc.Params.MaxLevel()
	// maxStartCipherLevel := 2

	for index := 0; index < len(convIDs); index++ {
		convID := convIDs[index]

		//register index of rotation
		rots := modules.MulParConvRegister(convID)

		//rotation key register
		newEvaluator := rotIndexToGaloisEl(int2dTo1d(rots), cc.Params, cc.Kgen, cc.Sk)

		//make mulParConv instance
		conv := modules.NewMulParConv(newEvaluator, cc.Encoder, cc.Params, convID)

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

				// MSE, RE, inf Norm
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
	rot := modules.ParBSGSFCRegister()

	//rot register
	newEvaluator := RotIndexToGaloisElements(rot, cc)

	//make avgPooling instance
	fc := modules.NewParBSGSFC(newEvaluator, cc.Encoder, cc.Params, 20)

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

	scores := MSE_RE_infNorm_1D(outputFloat[0:10], trueOutputFloat)
	fmt.Printf("MSE (Mean Squared Error)   : %.2e\n", scores[0])
	fmt.Printf("Relative Error             : %.2e\n", scores[1])
	fmt.Printf("Infinity Norm (L-infinity) : %.2e\n", scores[2])
}

func mulParfullyConnectedAccuracyTest(cc *customContext) {
	fmt.Println("Fully Connected + Conventional BSGS diagonal matrix-vector multiplication Test!")
	startLevel := 1
	endLevel := cc.Params.MaxLevel()
	//register
	rot := modules.MulParFCRegister()

	//rot register
	newEvaluator := RotIndexToGaloisElements(rot, cc)

	//make avgPooling instance
	fc := modules.NewMulParFC(newEvaluator, cc.Encoder, cc.Params, 20)

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

	scores := MSE_RE_infNorm_1D(outputFloat[0:10], trueOutputFloat)
	fmt.Printf("MSE (Mean Squared Error)   : %.2e\n", scores[0])
	fmt.Printf("Relative Error             : %.2e\n", scores[1])
	fmt.Printf("Infinity Norm (L-infinity) : %.2e\n", scores[2])
}

func rotOptDownSamplingTest(cc *customContext) {
	fmt.Println("Rotation Optimized Downsampling Test started! ")
	//register
	rot := modules.RotOptDSRegister()

	//rot register
	newEvaluator := RotIndexToGaloisElements(rot, cc)

	//make avgPooling instance
	ds16 := modules.NewRotOptDS(16, newEvaluator, cc.Encoder, cc.Params)
	ds32 := modules.NewRotOptDS(32, newEvaluator, cc.Encoder, cc.Params)

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
	rot := modules.MulParDSRegister()

	//rot register
	newEvaluator := RotIndexToGaloisElements(rot, cc)

	//make avgPooling instance
	ds16 := modules.NewMulParDS(16, newEvaluator, cc.Encoder, cc.Params)
	ds32 := modules.NewMulParDS(32, newEvaluator, cc.Encoder, cc.Params)

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

// Refer lattigo latest official document : https://pkg.go.dev/github.com/tuneinsight/lattigo/v4@v4.1.1/ckks#section-readme
func setCKKSEnvUseParamSet(paramSet string) *customContext {
	context := new(customContext)

	switch paramSet {
	case "PN16QP1761": // PN16QP1761 is a default parameter set for logN=16 and logQP = 1761
		context.Params, _ = ckks.NewParametersFromLiteral(ckks.ParametersLiteral{
			LogN: 16,
			Q: []uint64{0x80000000080001, 0x2000000a0001, 0x2000000e0001, 0x1fffffc20001,
				0x200000440001, 0x200000500001, 0x200000620001, 0x1fffff980001,
				0x2000006a0001, 0x1fffff7e0001, 0x200000860001, 0x200000a60001,
				0x200000aa0001, 0x200000b20001, 0x200000c80001, 0x1fffff360001,
				0x200000e20001, 0x1fffff060001, 0x200000fe0001, 0x1ffffede0001,
				0x1ffffeca0001, 0x1ffffeb40001, 0x200001520001, 0x1ffffe760001,
				0x2000019a0001, 0x1ffffe640001, 0x200001a00001, 0x1ffffe520001,
				0x200001e80001, 0x1ffffe0c0001, 0x1ffffdee0001, 0x200002480001,
				0x1ffffdb60001, 0x200002560001},
			P:               []uint64{0x80000000440001, 0x7fffffffba0001, 0x80000000500001, 0x7fffffffaa0001},
			LogDefaultScale: 45,
		})
	case "PN15QP880CI": // PN16QP1761CI is a default parameter set for logN=16 and logQP = 1761
		context.Params, _ = ckks.NewParametersFromLiteral(ckks.ParametersLiteral{
			LogN: 15,
			Q: []uint64{0x4000000120001,
				0x10000140001, 0xffffe80001, 0xffffc40001,
				0x100003e0001, 0xffffb20001, 0x10000500001,
				0xffff940001, 0xffff8a0001, 0xffff820001,
				0xffff780001, 0x10000960001, 0x10000a40001,
				0xffff580001, 0x10000b60001, 0xffff480001,
				0xffff420001, 0xffff340001},
			P:               []uint64{0x3ffffffd20001, 0x4000000420001, 0x3ffffffb80001},
			RingType:        ring.ConjugateInvariant,
			LogDefaultScale: 40,
		})
	case "PN16QP1654pq": // PN16QP1654pq is a default (post quantum) parameter set for logN=16 and logQP=1654
		context.Params, _ = ckks.NewParametersFromLiteral(ckks.ParametersLiteral{
			LogN: 16,
			Q: []uint64{0x80000000080001, 0x2000000a0001, 0x2000000e0001, 0x1fffffc20001, 0x200000440001,
				0x200000500001, 0x200000620001, 0x1fffff980001, 0x2000006a0001, 0x1fffff7e0001,
				0x200000860001, 0x200000a60001, 0x200000aa0001, 0x200000b20001, 0x200000c80001,
				0x1fffff360001, 0x200000e20001, 0x1fffff060001, 0x200000fe0001, 0x1ffffede0001,
				0x1ffffeca0001, 0x1ffffeb40001, 0x200001520001, 0x1ffffe760001, 0x2000019a0001,
				0x1ffffe640001, 0x200001a00001, 0x1ffffe520001, 0x200001e80001, 0x1ffffe0c0001,
				0x1ffffdee0001, 0x200002480001},
			P:               []uint64{0x7fffffffe0001, 0x80000001c0001, 0x80000002c0001, 0x7ffffffd20001},
			LogDefaultScale: 45,
		})
	case "PN15QP827CIpq": // PN16QP1654CIpq is a default (post quantum) parameter set for logN=16 and logQP=1654
		context.Params, _ = ckks.NewParametersFromLiteral(ckks.ParametersLiteral{
			LogN: 15,
			Q: []uint64{0x400000060001, 0x3fffe80001, 0x4000300001, 0x3fffb80001,
				0x40004a0001, 0x3fffb20001, 0x4000540001, 0x4000560001,
				0x3fff900001, 0x4000720001, 0x3fff8e0001, 0x4000800001,
				0x40008a0001, 0x3fff6c0001, 0x40009e0001, 0x3fff300001,
				0x3fff1c0001, 0x4000fc0001},
			P:               []uint64{0x2000000a0001, 0x2000000e0001, 0x1fffffc20001},
			RingType:        ring.ConjugateInvariant,
			LogDefaultScale: 38,
		})
	default:
		fmt.Printf("Unsupported CKKS parameter set name : %s\n", paramSet)
		fmt.Printf("CKKS setting set as default")
		return setCKKSEnv()
	}

	fmt.Printf("CKKS parameter set as : %s\n", paramSet)

	return setCKKSContext(context)
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
	fmt.Printf("CKKS parameter set as : default\n")
	return setCKKSContext(context)
}

func setCKKSContext(context *customContext) *customContext {
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
		mulParRot := modules.MulParConvRegister(convIDs[index])
		rotOptRot := modules.RotOptConvRegister(convIDs[index], maxDepth[index])

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
	fmt.Println("You can test other blue prints in modules/convConfig.go")

	convIDs := []string{"CONV1", "CONV2", "CONV3s2", "CONV3", "CONV4s2", "CONV4", "CvTCifar100Stage2", "CvTCifar100Stage3", "MUSE_PyramidGenConv"}
	maxDepth := []int{2, 4, 5, 4, 5, 4, 5, 6, 6}

	for index := 0; index < len(convIDs); index++ {
		for depth := 2; depth <= maxDepth[index]; depth++ {
			fmt.Printf("=== convID : %s, depth : %v === \n", convIDs[index], depth)
			convMap, _, _ := modules.GetConvBlueprints(convIDs[index], depth)
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
			for _, row := range modules.GetMulParConvFeature(convIDs[index]).KernelBP {
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
	fmt.Println("\nConventional BSGS diagonal matrix-vector multiplication Test!")
	fmt.Println("matrix : ", N, "x", N, "  vector : ", N, "x", 1)
	nt := 32768

	fmt.Printf("=== Conventional (BSGS diag mat(N*N)-vec(N*1) mul) method start! N : %v ===\n", N)

	A := getMatrix(N, N)
	B := getMatrix(N, 1)

	//answer
	answer := originalMatMul(A, B)

	//change B to ciphertext
	B1d := make2dTo1d(B)
	B1d = resize(B1d, nt)
	//start mat vec mul
	rot := modules.BsgsDiagMatVecMulRegister(N)
	newEvaluator := RotIndexToGaloisElements(rot, cc)
	matVecMul := modules.NewBsgsDiagMatVecMul(A, N, nt, newEvaluator, cc.Encoder, cc.Params)

	fmt.Printf("startLevel executionTime\n")
	var MSEList, REList, infNormList []float64
	for level := 1; level <= cc.Params.MaxLevel(); level++ {

		Bct := floatToCiphertextLevel(B1d, level, cc.Params, cc.Encoder, cc.EncryptorSk)

		startTime := time.Now()
		BctOut := matVecMul.Foward(Bct)
		endTime := time.Now()
		outputFloat := ciphertextToFloat(BctOut, cc)

		scores := MSE_RE_infNorm_1D(outputFloat[0:N], make2dTo1d(answer))
		MSEList = append(MSEList, scores[0])
		REList = append(REList, scores[1])
		infNormList = append(infNormList, scores[2])

		fmt.Println(level, endTime.Sub(startTime))
	}
	MSEMin, MSEMax, MSEAvg := minMaxAvg(MSEList)
	REMin, REMax, REAvg := minMaxAvg(REList)
	infNormMin, infNormMax, infNormAvg := minMaxAvg(infNormList)

	fmt.Printf("MSE (Mean Squared Error)   : Min = %.2e, Max = %.2e, Avg = %.2e\n", MSEMin, MSEMax, MSEAvg)
	fmt.Printf("Relative Error             : Min = %.2e, Max = %.2e, Avg = %.2e\n", REMin, REMax, REAvg)
	fmt.Printf("Infinity Norm (L-infinity) : Min = %.2e, Max = %.2e, Avg = %.2e\n", infNormMin, infNormMax, infNormAvg)
}
func parBsgsMatVecMultAccuracyTest(N int, cc *customContext) {
	fmt.Println("\nParallel BSGS matrix-vector multiplication Test!")
	fmt.Println("matrix : ", N, "x", N, "  vector : ", N, "x", 1)
	nt := cc.Params.MaxSlots()

	pi := 1 //initially setting. (how many identical datas are in single ciphertext)

	fmt.Printf("=== Proposed (Parallely BSGS diag mat(N*N)-vec(N*1) mul) method start! N : %v ===\n", N)

	A := getMatrix(N, N)
	B := getMatrix(N, 1)

	answer := originalMatMul(A, B)

	B1d := make2dTo1d(B)
	B1d = resize(B1d, nt)
	for i := 1; i < pi; i *= 2 {
		tempB := rotate(B1d, -(nt/pi)*i)
		B1d = add(tempB, B1d)
	}
	//start mat vec mul
	rot := modules.ParBsgsDiagMatVecMulRegister(N, nt, pi)
	newEvaluator := RotIndexToGaloisElements(rot, cc)
	matVecMul := modules.NewParBsgsDiagMatVecMul(A, N, nt, pi, newEvaluator, cc.Encoder, cc.Params)

	fmt.Printf("startLevel executionTime\n")
	var MSEList, REList, infNormList []float64
	for level := 1; level <= cc.Params.MaxLevel(); level++ {

		Bct := floatToCiphertextLevel(B1d, level, cc.Params, cc.Encoder, cc.EncryptorSk)

		startTime := time.Now()
		BctOut := matVecMul.Foward(Bct)
		endTime := time.Now()
		outputFloat := ciphertextToFloat(BctOut, cc)

		scores := MSE_RE_infNorm_1D(outputFloat[0:N], make2dTo1d(answer))
		MSEList = append(MSEList, scores[0])
		REList = append(REList, scores[1])
		infNormList = append(infNormList, scores[2])

		fmt.Println(level, endTime.Sub(startTime))
	}
	MSEMin, MSEMax, MSEAvg := minMaxAvg(MSEList)
	REMin, REMax, REAvg := minMaxAvg(REList)
	infNormMin, infNormMax, infNormAvg := minMaxAvg(infNormList)

	fmt.Printf("MSE (Mean Squared Error)   : Min = %.2e, Max = %.2e, Avg = %.2e\n", MSEMin, MSEMax, MSEAvg)
	fmt.Printf("Relative Error             : Min = %.2e, Max = %.2e, Avg = %.2e\n", REMin, REMax, REAvg)
	fmt.Printf("Infinity Norm (L-infinity) : Min = %.2e, Max = %.2e, Avg = %.2e\n", infNormMin, infNormMax, infNormAvg)
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
