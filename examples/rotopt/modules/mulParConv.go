package modules

import (
	"github.com/tuneinsight/lattigo/v5/core/rlwe"
	"github.com/tuneinsight/lattigo/v5/schemes/ckks"
)

// ///////////////////////////////////////////////////////////////////////////////////////////////////////
// Conventional multiplexed parallel convolution
// ///////////////////////////////////////////////////////////////////////////////////////////////////////
type MulParConv struct {
	encoder *ckks.Encoder

	Evaluator      *ckks.Evaluator
	params         ckks.Parameters
	PreCompKernels [][]*rlwe.Plaintext
	preCompFilters [][]*rlwe.Plaintext

	ConvFeature *ConvFeature

	q                  int //length of kernel_map
	rotIndex3by3Kernel []int
	depth1Rotate       []int
	depth0Rotate       []int
}

func NewMulParConv(ev *ckks.Evaluator, ec *ckks.Encoder, params ckks.Parameters, convID string) *MulParConv {
	// ("Conv : ", resnetLayerNum, convID, depth, blockNum, operationNum)

	//MulParConv Setting
	_, q, rotIndex3by3Kernel := GetConvBlueprints(convID, 2)

	// conv feature
	cf := GetMulParConvFeature(convID)

	// plaintext setting, kernel weight
	var preCompFilter []*rlwe.Plaintext

	// preCompFilter generate
	preCompFilter = make([]*rlwe.Plaintext, cf.BeforeCopy)
	luFilter := LeftUpFilter(cf)
	if cf.Stride != 1 {
		luFilter = multVec(luFilter, StrideFilter(cf.K, cf.InputDataWidth))
	}
	spFilter := crossFilter(luFilter, cf.BeforeCopy)
	for i := 0; i < cf.BeforeCopy; i++ {
		preCompFilter[i] = ckks.NewPlaintext(params, params.MaxLevel())
		ec.Encode(spFilter[i], preCompFilter[i])
	}

	//depth1Rotate generate
	var depth1Rotate []int
	k := cf.K
	for ki := 1; ki < cf.K; ki *= 2 {
		depth1Rotate = append(depth1Rotate, ki)
	}

	for ki := 1; ki < cf.K; ki *= 2 {
		depth1Rotate = append(depth1Rotate, cf.InputDataWidth*k*ki)
	}

	for bi := 1; bi < cf.InputDataChannel/cf.K/cf.K; bi *= 2 {
		depth1Rotate = append(depth1Rotate, cf.InputDataWidth*cf.InputDataWidth*k*k*bi)
	}

	//depth0Rotate generate
	var depth0Rotate []int

	for inputChannel := 0; inputChannel < cf.KernelNumber; inputChannel++ {
		beforeLocate := getFirstLocate(0, inputChannel%cf.BeforeCopy, cf, true)

		afterLoate := getFirstLocate(inputChannel, 0, cf, false)
		depth0Rotate = append(depth0Rotate, beforeLocate-afterLoate)
	}

	//PerRotate preCompFilter to minimze rescaling
	preCompFilters := make([][]*rlwe.Plaintext, cf.q)
	for cipherNum := 0; cipherNum < cf.q; cipherNum++ {
		for eachCopy := 0; eachCopy < cf.BeforeCopy; eachCopy++ {
			preCompFilters[cipherNum] = append(preCompFilters[cipherNum], PlaintextRot(preCompFilter[eachCopy], depth0Rotate[cipherNum*cf.BeforeCopy+eachCopy], ec, params))
		}
	}

	return &MulParConv{
		encoder: ec,

		Evaluator:      ev,
		params:         params,
		preCompFilters: preCompFilters,
		ConvFeature:    cf,

		q:                  q,
		rotIndex3by3Kernel: rotIndex3by3Kernel,
		depth0Rotate:       depth0Rotate,
		depth1Rotate:       depth1Rotate,
	}
}

func (obj MulParConv) Foward(ctIn *rlwe.Ciphertext) (ctOut *rlwe.Ciphertext) {

	rotnum := 0

	mainCipher := ckks.NewCiphertext(obj.params, 1, ctIn.Level())
	tempCtLv1 := ckks.NewCiphertext(obj.params, 1, ctIn.Level())
	tempCtLv0 := ckks.NewCiphertext(obj.params, 1, ctIn.Level())

	var err error
	// start := time.Now()

	// Rotate Data
	var rotInput []*rlwe.Ciphertext
	for w := 0; w < 9; w++ {
		c, err := obj.Evaluator.RotateNew(ctIn, obj.rotIndex3by3Kernel[w])

		ErrorPrint(err)
		rotInput = append(rotInput, c)
	}
	rotnum--
	// fmt.Println("rotate data ", time.Now().Sub(start))

	//For each ciphertext
	for cipherNum := 0; cipherNum < obj.ConvFeature.q; cipherNum++ {
		// Mul kernels
		kernelResult, err := obj.Evaluator.MulNew(rotInput[0], obj.PreCompKernels[cipherNum][0])
		ErrorPrint(err)
		// err = obj.Evaluator.Rescale(tempCt, tempCt)
		// ErrorPrint(err)

		for w := 1; w < 9; w++ {
			tempCt, err := obj.Evaluator.MulNew(rotInput[w], obj.PreCompKernels[cipherNum][w])
			ErrorPrint(err)
			// err = obj.Evaluator.Rescale(tempCt, tempCtLv1)
			// ErrorPrint(err)
			err = obj.Evaluator.Add(kernelResult, tempCt, kernelResult)
			ErrorPrint(err)
		}

		err = obj.Evaluator.Rescale(kernelResult, mainCipher)
		ErrorPrint(err)

		//left up
		for rotLeftUp := 0; rotLeftUp < len(obj.depth1Rotate); rotLeftUp++ {

			err = obj.Evaluator.Rotate(mainCipher, obj.depth1Rotate[rotLeftUp], tempCtLv1)
			ErrorPrint(err)

			err = obj.Evaluator.Add(mainCipher, tempCtLv1, mainCipher)
			ErrorPrint(err)
		}

		//Mul each filter to get each channel
		for eachCopy := 0; eachCopy < obj.ConvFeature.BeforeCopy; eachCopy++ {
			if cipherNum == 0 && eachCopy == 0 {
				temp, err := obj.Evaluator.RotateNew(mainCipher, obj.depth0Rotate[cipherNum*obj.ConvFeature.BeforeCopy+eachCopy])
				ErrorPrint(err)
				ctOut, _ = obj.Evaluator.MulNew(temp, obj.preCompFilters[cipherNum][eachCopy])
			} else {
				temp, err := obj.Evaluator.RotateNew(mainCipher, obj.depth0Rotate[cipherNum*obj.ConvFeature.BeforeCopy+eachCopy])
				ErrorPrint(err)
				obj.Evaluator.Mul(temp, obj.preCompFilters[cipherNum][eachCopy], temp)
				ErrorPrint(err)
				err = obj.Evaluator.Add(ctOut, temp, ctOut)
				ErrorPrint(err)
			}
		}

	}

	obj.Evaluator.Rescale(ctOut, ctOut)

	for afterCopy := 32768 / obj.ConvFeature.AfterCopy; afterCopy < 32768; afterCopy *= 2 {
		obj.Evaluator.Rotate(ctOut, -afterCopy, tempCtLv0)
		obj.Evaluator.Add(ctOut, tempCtLv0, ctOut)
	}

	//Add bn_add
	// ctOut, err = obj.Evaluator.AddNew(ctOut, obj.preCompBNadd)
	ErrorPrint(err)

	return ctOut
}

func MulParConvRegister(convID string) [][]int {
	rotateSets := make([]map[int]bool, 3)

	for d := 0; d < 3; d++ {
		rotateSets[d] = make(map[int]bool)
	}

	_, _, rotIndex3by3Kernel := GetConvBlueprints(convID, 2)

	//Depth 2
	for i := 0; i < len(rotIndex3by3Kernel); i++ {
		rotateSets[2][rotIndex3by3Kernel[i]] = true
	}

	//Depth1
	ConvFeature := GetMulParConvFeature(convID)
	k := ConvFeature.K

	for ki := 1; ki < k; ki *= 2 {
		rotateSets[1][ki] = true
		rotateSets[1][ConvFeature.InputDataWidth*k*ki] = true
	}

	for bi := 1; bi < ConvFeature.InputDataChannel/k/k; bi *= 2 {
		rotateSets[1][ConvFeature.InputDataWidth*ConvFeature.InputDataWidth*k*k*bi] = true
	}

	//Depth0
	for inputChannel := 0; inputChannel < ConvFeature.KernelNumber; inputChannel++ {
		beforeLocate := getFirstLocate(0, inputChannel%ConvFeature.BeforeCopy, ConvFeature, true)

		afterLoate := getFirstLocate(inputChannel, 0, ConvFeature, false)

		rotateSets[0][beforeLocate-afterLoate] = true
	}
	for afterCopy := 32768 / ConvFeature.AfterCopy; afterCopy < 32768; afterCopy *= 2 {
		rotateSets[0][-afterCopy] = true
	}

	rotateArray := make([][]int, 3)
	for d := 0; d < 3; d++ {
		rotateArray[d] = make([]int, 0)
		for element := range rotateSets[d] {
			if element != 0 {
				rotateArray[d] = append(rotateArray[d], element)
			}
		}
	}

	return rotateArray

}

func getFirstLocate(channel int, sameCopy int, cf *ConvFeature, isBefore bool) int {
	ctLen := 32768

	k := cf.AfterK
	w := cf.InputDataWidth / cf.Stride
	h := cf.InputDataHeight / cf.Stride
	if isBefore {
		k = cf.K
		w = cf.InputDataWidth
		h = cf.InputDataHeight
	}
	locate := channel%k + channel%(k*k)/k*w*k + channel/(k*k)*w*h*k*k + (ctLen/cf.BeforeCopy)*sameCopy

	return locate
}
