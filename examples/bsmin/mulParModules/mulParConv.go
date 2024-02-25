package mulParModules

import (
	"fmt"
	"strconv"

	"github.com/tuneinsight/lattigo/v5/core/rlwe"
	"github.com/tuneinsight/lattigo/v5/schemes/ckks"
	"github.com/tuneinsight/lattigo/v5/utils/sampling"
)

type MulParConv struct {
	//for debugging
	encoder   *ckks.Encoder
	decryptor *rlwe.Decryptor

	Evaluator      *ckks.Evaluator
	params         ckks.Parameters
	preCompKernel  [][]*rlwe.Plaintext
	preCompBNadd   *rlwe.Plaintext
	preCompFilter  [][]*rlwe.Plaintext
	mode0TreeDepth int
	cf             *ConvFeature

	layerNum           int
	blockNum           int
	operationNum       int
	convMap            [][]int
	q                  int //length of kernel_map
	rotIndex3by3Kernel []int
	beforeSplitNum     int
	splitNum           int
}

func NewMulParConv(ev *ckks.Evaluator, ec *ckks.Encoder, dc *rlwe.Decryptor, params ckks.Parameters, resnetLayerNum int, convID string, depth int, blockNum int, operationNum int) *MulParConv {
	// fmt.Println("Conv : ", resnetLayerNum, convID, depth, blockNum, operationNum)

	//MulParConv Setting
	convMap, q, rotIndex3by3Kernel := GetConvMap(convID, depth)

	// conv feature
	cf := GetConvFeature(convID)

	// plaintext setting, kernel weight
	path := "mulParModules/precomputed/rotOptConv/kernelWeight/" + strconv.Itoa(resnetLayerNum) + "/" + cf.LayerStr + "/" + strconv.Itoa(blockNum) + "/"
	var preCompKernel [][]*rlwe.Plaintext
	var preCompBNadd *rlwe.Plaintext
	var preCompFilter [][]*rlwe.Plaintext

	// preCompKernel generate
	filePath := path + "conv" + strconv.Itoa(operationNum) + "_weight"
	for i := 0; i < len(cf.KernelMap); i++ {
		var temp []*rlwe.Plaintext
		for j := 0; j < 9; j++ {
			temp = append(temp, txtToPlain(ec, filePath+strconv.Itoa(i)+"_"+strconv.Itoa(j)+".txt", params))
		}
		preCompKernel = append(preCompKernel, temp)
	}

	// preCompBNadd generate
	filePath = path + "bn" + strconv.Itoa(operationNum) + "_add.txt"
	preCompBNadd = txtToPlain(ec, filePath, params)

	// preCompFilter generate
	preCompFilter = make([][]*rlwe.Plaintext, cf.q)
	for i := 0; i < cf.q; i++ {
		preCompFilter[i] = make([]*rlwe.Plaintext, cf.BeforeCopy)
		for j := 0; j < cf.BeforeCopy; j++ {
			preCompFilter[i][j] = ckks.NewPlaintext(params, 2)
			ec.Encode(makeRandomFloat(params.MaxSlots()), preCompFilter[i][j])
		}
	}

	return &MulParConv{
		encoder:   ec,
		decryptor: dc,

		Evaluator:     ev,
		params:        params,
		preCompKernel: preCompKernel,
		preCompBNadd:  preCompBNadd,
		preCompFilter: preCompFilter,
		cf:            cf,

		layerNum:           resnetLayerNum,
		blockNum:           blockNum,
		operationNum:       operationNum,
		convMap:            convMap,
		q:                  q,
		rotIndex3by3Kernel: rotIndex3by3Kernel,
	}
}

//for debugging

func (obj MulParConv) printCipher(fileName string, ctIn *rlwe.Ciphertext) {

	plainIn := obj.decryptor.DecryptNew(ctIn)
	floatIn := make([]float64, obj.params.MaxSlots())
	obj.encoder.Decode(plainIn, floatIn)

	floatToTxt(fileName+".txt", floatIn)

}

func (obj MulParConv) Foward(ctIn *rlwe.Ciphertext) (ctOut *rlwe.Ciphertext) {

	rotnum := 0

	mainCipher := ckks.NewCiphertext(obj.params, 1, ctIn.Level())
	tempCtLv1 := ckks.NewCiphertext(obj.params, 1, ctIn.Level())
	// tempCtLv0 := ckks.NewCiphertext(obj.params, 1, ctIn.Level())

	var err error

	// Rotate Data
	var rotInput []*rlwe.Ciphertext
	for w := 0; w < 9; w++ {
		c, err := obj.Evaluator.RotateNew(ctIn, obj.rotIndex3by3Kernel[w])
		rotnum++
		ErrorPrint(err)
		rotInput = append(rotInput, c)
	}
	rotnum--

	//For each ciphertext
	for cipherNum := 0; cipherNum < obj.cf.q; cipherNum++ {
		// Mul kernels (후에 커널 구조 수정)
		kernelResult, err := obj.Evaluator.MulNew(rotInput[0], obj.preCompKernel[cipherNum][0])
		ErrorPrint(err)
		// err = obj.Evaluator.Rescale(tempCt, tempCt)
		// ErrorPrint(err)

		// mainCipher = tempCt

		for w := 1; w < 9; w++ {
			tempCt, err := obj.Evaluator.MulNew(rotInput[w], obj.preCompKernel[cipherNum][w])
			ErrorPrint(err)
			// err = obj.Evaluator.Rescale(tempCt, tempCtLv1)
			// ErrorPrint(err)
			err = obj.Evaluator.Add(kernelResult, tempCt, kernelResult)
			ErrorPrint(err)
		}

		err = obj.Evaluator.Rescale(kernelResult, mainCipher)
		ErrorPrint(err)

		//left up
		for rotLeftUp := 1; rotLeftUp < obj.cf.InputDataChannel; rotLeftUp *= 2 {

			err = obj.Evaluator.Rotate(mainCipher, rotLeftUp, tempCtLv1)
			ErrorPrint(err)
			rotnum++
			err = obj.Evaluator.Add(mainCipher, tempCtLv1, mainCipher)
			ErrorPrint(err)
		}

		//Mul each filter to get each channel
		for eachCopy := 0; eachCopy < obj.cf.BeforeCopy; eachCopy++ {
			tempRelin, _ := obj.Evaluator.MulNew(mainCipher, obj.preCompFilter[cipherNum][eachCopy])
			obj.Evaluator.Rescale(tempRelin, tempRelin) // 이거 없앨수있나..?
			if cipherNum*eachCopy+eachCopy == 0 {
				ctOut, err = obj.Evaluator.RotateNew(tempRelin, cipherNum*eachCopy+eachCopy)
				ErrorPrint(err)
				rotnum++
			} else {
				obj.Evaluator.Rotate(tempRelin, cipherNum*eachCopy+eachCopy, tempRelin)
				obj.Evaluator.Add(ctOut, tempRelin, ctOut)
				rotnum++
			}
		}
	}

	for afterCopy := 1; afterCopy < obj.cf.AfterCopy; afterCopy *= 2 {
		obj.Evaluator.Rotate(ctOut, afterCopy, ctOut)
		rotnum++
	}
	fmt.Println(rotnum)
	//Add bn_add
	ctOut, err = obj.Evaluator.AddNew(ctOut, obj.preCompBNadd)
	ErrorPrint(err)

	return ctOut
}

func MulParConvRegister(convID string, depth int) []int {
	var rotIndex []int
	_, _, rotIndex3by3Kernel := GetConvMap(convID, depth)
	for i := 0; i < GetConvFeature(convID).KernelNumber; i++ {
		rotIndex = append(rotIndex, i)
	}

	for i := 0; i < len(rotIndex3by3Kernel); i++ {
		rotIndex = append(rotIndex, rotIndex3by3Kernel[i])
	}

	return rotIndex

}

func makeRandomFloat(length int) []float64 {
	valuesWant := make([]float64, length)
	for i := range valuesWant {
		valuesWant[i] = sampling.RandFloat64(-1, 1)
	}
	return valuesWant
}
