package main

import (
	"fmt"
	"rotOptResnet/mulParModules"

	"github.com/tuneinsight/lattigo/v5/core/rlwe"
	"github.com/tuneinsight/lattigo/v5/schemes/ckks"
)

func rotOptConvAccuracyTest(layerNum int, cc *customContext, convID string, convDepth int, startCipherLevel int) {

	fmt.Printf("\n=== RotOptConvAccuracyTest. convID : %s, Depth : %v === \n", convID, startCipherLevel)
	cf := mulParModules.GetConvFeature(convID)

	////True logs////
	inputTxtPath, outputTxtPath := getTrueConvTestTxtPath(convID)

	inputTrueLogs := txtToFloat(inputTxtPath)
	outputTrueLogs := txtToFloat(outputTxtPath)

	if convID == "CONV1" {
		for i := 0; i < 1024; i++ {
			inputTrueLogs = append(inputTrueLogs, 0)
		}
	}

	var modifiedInputTrueLogs []float64
	var modifiedOutputTrueLogs []float64

	modifiedInputTrueLogs = packing(inputTrueLogs, cf.K)
	modifiedInputTrueLogs = copyPaste(modifiedInputTrueLogs, 32768/len(modifiedInputTrueLogs))
	modifiedOutputTrueLogs = packing(outputTrueLogs, cf.AfterK)
	modifiedOutputTrueLogs = copyPaste(modifiedOutputTrueLogs, 32768/len(modifiedOutputTrueLogs))

	plain := ckks.NewPlaintext(cc.Params, startCipherLevel)
	cc.Encoder.Encode(modifiedInputTrueLogs, plain)
	inputCt, _ := cc.EncryptorSk.EncryptNew(plain)

	//register
	rots := mulParModules.RotOptConvRegister(convID, convDepth)

	//rot register
	newEvaluator := rotIndexToGaloisEl(int2dTo1d(rots), cc.Params, cc.Kgen, cc.Sk)

	//make rotOptConv instance
	conv := mulParModules.NewrotOptConv(newEvaluator, cc.Encoder, cc.Decryptor, cc.Params, layerNum, convID, convDepth, getConvTestNum(convID)[0], getConvTestNum(convID)[1])

	var outputCt *rlwe.Ciphertext

	//Conv Foward
	outputCt = conv.Foward(inputCt)

	//Decryption
	outputFloat := ciphertextToFloat(outputCt, cc)

	fmt.Println("Euclidean Distance Accuracy:", euclideanDistance(outputFloat, modifiedOutputTrueLogs))
}

func mulParConvAccuracyTest(layerNum int, cc *customContext, convID string, startCipherLevel int) {

	fmt.Printf("\n=== MulParConvAccuracyTest. convID : %s, Depth : %v === \n", convID, startCipherLevel)
	cf := mulParModules.GetConvFeature(convID)

	////True logs////
	inputTxtPath, outputTxtPath := getTrueConvTestTxtPath(convID)

	inputTrueLogs := txtToFloat(inputTxtPath)
	outputTrueLogs := txtToFloat(outputTxtPath)

	if convID == "CONV1" {
		for i := 0; i < 1024; i++ {
			inputTrueLogs = append(inputTrueLogs, 0)
		}
	}

	var modifiedInputTrueLogs []float64
	var modifiedOutputTrueLogs []float64

	modifiedInputTrueLogs = packing(inputTrueLogs, cf.K)
	modifiedInputTrueLogs = copyPaste(modifiedInputTrueLogs, 32768/len(modifiedInputTrueLogs))
	modifiedOutputTrueLogs = packing(outputTrueLogs, cf.AfterK)
	modifiedOutputTrueLogs = copyPaste(modifiedOutputTrueLogs, 32768/len(modifiedOutputTrueLogs))

	plain := ckks.NewPlaintext(cc.Params, startCipherLevel)
	cc.Encoder.Encode(modifiedInputTrueLogs, plain)
	inputCt, _ := cc.EncryptorSk.EncryptNew(plain)

	//register
	rots := mulParModules.MulParConvRegister(convID)

	//rot register
	newEvaluator := rotIndexToGaloisEl(int2dTo1d(rots), cc.Params, cc.Kgen, cc.Sk)

	//make rotOptConv instance
	conv := mulParModules.NewMulParConv(newEvaluator, cc.Encoder, cc.Decryptor, cc.Params, layerNum, convID, getConvTestNum(convID)[0], getConvTestNum(convID)[1])

	var outputCt *rlwe.Ciphertext

	//Conv Foward
	outputCt = conv.Foward(inputCt)

	//Decryption
	outputFloat := ciphertextToFloat(outputCt, cc)

	fmt.Println("Euclidean Distance Accuracy:", euclideanDistance(outputFloat, modifiedOutputTrueLogs))
}

func getTrueConvTestTxtPath(convID string) (string, string) {
	folderName := "true_logs/"
	if convID == "CONV1" {
		return folderName + "sample_data.txt", folderName + "layer0_bn.txt"
	} else if convID == "CONV2" {
		return folderName + "layer0_layerEnd.txt", folderName + "layer1_0_bn1.txt"
	} else if convID == "CONV3s2" {
		return folderName + "layer1_layerEnd.txt", folderName + "layer2_0_bn1.txt"
	} else if convID == "CONV3" {
		return folderName + "layer2_0_relu1.txt", folderName + "layer2_0_bn2.txt"
	} else if convID == "CONV4s2" {
		return folderName + "layer2_layerEnd.txt", folderName + "layer3_0_bn1.txt"
	} else if convID == "CONV4" {
		return folderName + "layer3_0_relu1.txt", folderName + "layer3_0_bn2.txt"
	}
	fmt.Println("Not existing convID :", convID)
	return "", ""
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
