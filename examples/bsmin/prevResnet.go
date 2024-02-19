package main

import (
	"fmt"
	"rotOptResnet/mulParModules"
	"strconv"

	"github.com/tuneinsight/lattigo/v5/core/rlwe"
	"github.com/tuneinsight/lattigo/v5/schemes/ckks"
)

type preBlock struct {
	preblockNumForLog  int
	prevlayerNumForLog int

	ResnetprevLayerNum int

	prevLayerStart int
	prevLayerEnd   int
	Planes         int
	Stride         int

	Convbn1 *mulParModules.MulParConv
	Relu1   *mulParModules.Relu
	Convbn2 *mulParModules.MulParConv
	Relu2   *mulParModules.Relu

	Downsampling *mulParModules.RotOptDS

	ConvDepthPlan []int

	Evaluator *ckks.Evaluator
	Encoder   *ckks.Encoder
	Decryptor *rlwe.Decryptor
	params    ckks.Parameters
}
type prevLayer struct {
	ResnetprevLayerNum int

	prevLayerNum   int
	prevLayerStart int
	prevLayerEnd   int
	Planes         int
	Stride         int

	preBlocks    []*preBlock
	preBlocksLen int

	ConvDepthPlan []int

	Evaluator *ckks.Evaluator
	Encoder   *ckks.Encoder
	Decryptor *rlwe.Decryptor
	params    ckks.Parameters
}
type prevResnetCifar10 struct {
	ResnetprevLayerNum int

	Convbn1        *mulParModules.MulParConv
	Relu1          *mulParModules.Relu
	prevLayer1     *prevLayer
	prevLayer2     *prevLayer
	prevLayer3     *prevLayer
	AvgPool        *mulParModules.AvgPool
	FullyConnected *mulParModules.ParFC

	Evaluator *ckks.Evaluator
	Encoder   *ckks.Encoder
	Decryptor *rlwe.Decryptor
	params    ckks.Parameters

	ConvDepthPlan []int
}

func NewpreBlock(resnetprevLayerNum int, prevLayerNum int, preBlockNum int, prevlayerStart int, prevlayerEnd int, planes int, stride int, ConvDepthPlan []int, Evaluator *ckks.Evaluator, Encoder *ckks.Encoder, Decryptor *rlwe.Decryptor, params ckks.Parameters, Encryptor *rlwe.Encryptor) *preBlock {
	var ds *mulParModules.RotOptDS
	if stride != 1 {
		ds = mulParModules.NewRotOptDS(planes/2, Evaluator, Encoder, params)
	} else {
		ds = nil
	}

	var convID1, convID2 string
	if prevLayerNum == 1 {
		convID1, convID2 = "CONV2", "CONV2"
	} else if prevLayerNum == 2 && stride == 2 {
		convID1, convID2 = "CONV3s2", "CONV3"
	} else if prevLayerNum == 2 && stride == 1 {
		convID1, convID2 = "CONV3", "CONV3"
	} else if prevLayerNum == 3 && stride == 2 {
		convID1, convID2 = "CONV4s2", "CONV4"
	} else if prevLayerNum == 3 && stride == 1 {
		convID1, convID2 = "CONV4", "CONV4"
	}

	fmt.Println(resnetprevLayerNum, convID1, ConvDepthPlan[prevlayerStart], preBlockNum, 1)
	fmt.Println(resnetprevLayerNum, convID2, ConvDepthPlan[prevlayerStart], preBlockNum, 2)

	return &preBlock{
		preblockNumForLog:  preBlockNum,
		prevlayerNumForLog: prevLayerNum,

		Convbn1: mulParModules.NewMulParConv(Evaluator, Encoder, Decryptor, params, resnetprevLayerNum, convID1, ConvDepthPlan[prevlayerStart], preBlockNum, 1),
		Relu1:   mulParModules.NewRelu(Evaluator, Encoder, Decryptor, Encryptor, params),
		Convbn2: mulParModules.NewMulParConv(Evaluator, Encoder, Decryptor, params, resnetprevLayerNum, convID2, ConvDepthPlan[prevlayerStart], preBlockNum, 2),
		Relu2:   mulParModules.NewRelu(Evaluator, Encoder, Decryptor, Encryptor, params),

		Downsampling:  ds,
		ConvDepthPlan: ConvDepthPlan,
		Evaluator:     Evaluator,

		prevLayerStart: prevlayerStart,

		Decryptor: Decryptor,
		params:    params,
		Encoder:   Encoder,
	}
}
func NewprevLayer(resnetprevLayerNum int, prevLayerNum int, prevlayerStart int, prevlayerEnd int, planes int, stride int, ConvDepthPlan []int, Evaluator *ckks.Evaluator, Encoder *ckks.Encoder, Decryptor *rlwe.Decryptor, params ckks.Parameters, Encryptor *rlwe.Encryptor) *prevLayer {
	containpreBlockNum := (prevlayerEnd - prevlayerStart + 1) / 2

	var preBlocks []*preBlock
	for i := 0; i < containpreBlockNum; i++ {
		if i == 0 && stride != 1 {
			preBlocks = append(preBlocks, NewpreBlock(resnetprevLayerNum, prevLayerNum, i, prevlayerStart+2*i, prevlayerStart+2*(i+1)-1, planes, stride, ConvDepthPlan, Evaluator, Encoder, Decryptor, params, Encryptor))
		} else {
			preBlocks = append(preBlocks, NewpreBlock(resnetprevLayerNum, prevLayerNum, i, prevlayerStart+2*i, prevlayerStart+2*(i+1)-1, planes, 1, ConvDepthPlan, Evaluator, Encoder, Decryptor, params, Encryptor))
		}
	}

	return &prevLayer{
		ConvDepthPlan: ConvDepthPlan,
		preBlocks:     preBlocks,
		prevLayerNum:  prevLayerNum,
		preBlocksLen:  containpreBlockNum,

		Decryptor: Decryptor,
		params:    params,
		Encoder:   Encoder,
	}
}
func NewprevResnetCifar10(resnetprevLayerNum int, Evaluator *ckks.Evaluator, Encoder *ckks.Encoder, Decryptor *rlwe.Decryptor, params ckks.Parameters, Encryptor *rlwe.Encryptor, kgen *rlwe.KeyGenerator, sk *rlwe.SecretKey) *prevResnetCifar10 {

	var convDepthPlan []int
	if resnetprevLayerNum == 20 {
		convDepthPlan = []int{
			2,
			2, 2, 2, 2, 2, 2,
			2, 2, 2, 2, 2, 2,
			2, 2, 2, 2, 2, 2,
		}
		// convDepthPlan = []int{
		// 	2,
		// 	3, 3, 3, 3, 3, 3,
		// 	3, 3, 3, 3, 3, 3,
		// 	3, 3, 3, 3, 3, 3,
		// }

	} else if resnetprevLayerNum == 32 {
		convDepthPlan = []int{
			2,
			2, 2, 2, 2, 2, 2,
			2, 2, 2, 2, 2, 2,
			2, 2, 2, 2, 2, 2,
			2, 2, 2, 2, 2, 2,
			2, 2, 2, 2, 2, 2,
		}
	}

	rotSet := make(map[int]bool)
	for i := -32768; i < 32769; i++ {
		rotSet[i] = false
	}
	//Conv1 Rot register
	rot := mulParModules.MulParConvRegister("CONV1", 2)
	rotSet = prevrotSetCombine(rotSet, rot)

	//prevlayer1 Rot Register
	depthCheck := make(map[int]bool)
	for i := 1; i < (resnetprevLayerNum-2)/3; i++ {
		convDepth := convDepthPlan[i]
		if !depthCheck[convDepth] {
			rot := mulParModules.MulParConvRegister("CONV2", convDepth)
			rotSet = prevrotSetCombine(rotSet, rot)
			depthCheck[convDepth] = true
		}
	}
	//prevlayer2 Rot Register
	rot = mulParModules.MulParConvRegister("CONV3s2", convDepthPlan[(resnetprevLayerNum-2)/3+1])
	rotSet = prevrotSetCombine(rotSet, rot)
	depthCheck = make(map[int]bool)
	for i := (resnetprevLayerNum-2)/3 + 1 + 1; i < 2*(resnetprevLayerNum-2)/3; i++ {
		convDepth := convDepthPlan[i]
		if !depthCheck[convDepth] {
			rot := mulParModules.MulParConvRegister("CONV3", convDepth)
			rotSet = prevrotSetCombine(rotSet, rot)
			depthCheck[convDepth] = true
		}
	}

	//prevlayer3 Rot Register
	rot = mulParModules.MulParConvRegister("CONV4s2", convDepthPlan[2*(resnetprevLayerNum-2)/3+1])
	rotSet = prevrotSetCombine(rotSet, rot)
	depthCheck = make(map[int]bool)
	for i := 2*(resnetprevLayerNum-2)/3 + 1 + 1; i < 3*(resnetprevLayerNum-2)/3; i++ {
		convDepth := convDepthPlan[i]
		if !depthCheck[convDepth] {
			rot := mulParModules.MulParConvRegister("CONV4", convDepth)
			rotSet = prevrotSetCombine(rotSet, rot)
			depthCheck[convDepth] = true
		}
	}
	// AvgPool rot register
	rot = mulParModules.AvgPoolRegister()
	rotSet = prevrotSetCombine(rotSet, rot)

	// FC rot register
	rot = mulParModules.ParFCRegister()
	rotSet = prevrotSetCombine(rotSet, rot)

	// DS rot register
	rot = mulParModules.RotOptDSRegister()
	rotSet = prevrotSetCombine(rotSet, rot)

	//change map to slice
	var trueIndices []int
	for index, value := range rotSet {
		if value {
			trueIndices = append(trueIndices, index)
		}
	}

	//Add to evaluator
	newEvaluator := prevrotIndexToGaloisEl(trueIndices, params, kgen, sk)

	return &prevResnetCifar10{
		ResnetprevLayerNum: resnetprevLayerNum,
		ConvDepthPlan:      convDepthPlan,
		prevLayer1:         NewprevLayer(resnetprevLayerNum, 1, 1, (resnetprevLayerNum-2)/3, 16, 1, convDepthPlan, newEvaluator, Encoder, Decryptor, params, Encryptor),
		prevLayer2:         NewprevLayer(resnetprevLayerNum, 2, (resnetprevLayerNum-2)/3+1, 2*(resnetprevLayerNum-2)/3, 32, 2, convDepthPlan, newEvaluator, Encoder, Decryptor, params, Encryptor),
		prevLayer3:         NewprevLayer(resnetprevLayerNum, 3, 2*(resnetprevLayerNum-2)/3+1, 3*(resnetprevLayerNum-2)/3, 64, 2, convDepthPlan, newEvaluator, Encoder, Decryptor, params, Encryptor),

		Convbn1:        mulParModules.NewMulParConv(newEvaluator, Encoder, Decryptor, params, resnetprevLayerNum, "CONV1", 2, 0, 1),
		Relu1:          mulParModules.NewRelu(newEvaluator, Encoder, Decryptor, Encryptor, params),
		AvgPool:        mulParModules.NewAvgPool(newEvaluator, Encoder, params),
		FullyConnected: mulParModules.NewparFC(newEvaluator, Encoder, params, resnetprevLayerNum),

		Decryptor: Decryptor,
		params:    params,
		Encoder:   Encoder,
	}
}

func (obj preBlock) Foward(ctIn *rlwe.Ciphertext) (ctOut *rlwe.Ciphertext) {

	tempCt := obj.Convbn1.Foward(ctIn)
	obj.myLogSave("prevlayer"+strconv.Itoa(obj.prevlayerNumForLog)+"_"+strconv.Itoa(obj.preblockNumForLog)+"_bn1", tempCt)
	tempCt = obj.Relu1.Foward(tempCt)
	tempCt = obj.Convbn2.Foward(tempCt)
	obj.myLogSave("prevlayer"+strconv.Itoa(obj.prevlayerNumForLog)+"_"+strconv.Itoa(obj.preblockNumForLog)+"_bn2", tempCt)

	if obj.Downsampling != nil {
		dsCt := obj.Downsampling.Foward(ctIn)
		obj.Evaluator.Add(tempCt, dsCt, tempCt)
	} else {
		obj.Evaluator.Add(tempCt, ctIn, tempCt)
	}

	ctOut = obj.Relu2.Foward(tempCt)

	obj.myLogSave(strconv.Itoa(obj.prevlayerNumForLog)+"_"+strconv.Itoa(obj.preblockNumForLog)+"preblockEnd", ctOut)

	return ctOut
}

func (obj prevLayer) Foward(ctIn *rlwe.Ciphertext) (ctOut *rlwe.Ciphertext) {

	tempCt := obj.preBlocks[0].Foward(ctIn)

	for b := 1; b < obj.preBlocksLen-1; b++ {
		tempCt = obj.preBlocks[b].Foward(tempCt)
	}

	ctOut = obj.preBlocks[obj.preBlocksLen-1].Foward(tempCt)
	return ctOut
}

func (obj prevResnetCifar10) Inference(ctIn *rlwe.Ciphertext) (ctOut *rlwe.Ciphertext) {
	tempCt := obj.Convbn1.Foward(ctIn)
	fmt.Println("after conv1", tempCt.Level(), tempCt.Scale)

	tempCt = obj.Relu1.Foward(tempCt)
	obj.myLogSave("prevlayer0End", tempCt)
	fmt.Println("prevlayer0End", tempCt.Level(), tempCt.Scale)

	tempCt = obj.prevLayer1.Foward(tempCt)
	obj.myLogSave("prevlayer1End", tempCt)
	fmt.Println("prevlayer1End", tempCt.Level(), tempCt.Scale)

	tempCt = obj.prevLayer2.Foward(tempCt)
	obj.myLogSave("prevlayer2End", tempCt)
	fmt.Println("prevlayer2End", tempCt.Level(), tempCt.Scale)

	tempCt = obj.prevLayer3.Foward(tempCt)
	obj.myLogSave("prevlayer3End", tempCt)
	fmt.Println("prevlayer3End", tempCt.Level(), tempCt.Scale)

	tempCt = obj.AvgPool.Foward(tempCt)
	obj.myLogSave("AvgPoolEnd", tempCt)
	ctOut = obj.FullyConnected.Foward(tempCt)
	obj.myLogSave("FcEnd", ctOut)
	return ctOut
}

func prevrotSetCombine(rotSet map[int]bool, rotIndices []int) map[int]bool {
	for i := 0; i < len(rotIndices); i++ {
		rotSet[rotIndices[i]] = true
	}
	return rotSet
}
func prevrotIndexToGaloisEl(input []int, params ckks.Parameters, kgen *rlwe.KeyGenerator, sk *rlwe.SecretKey) *ckks.Evaluator {
	var galElements []uint64

	for _, rotIndex := range input {
		galElements = append(galElements, params.GaloisElement(rotIndex))
	}
	galKeys := kgen.GenGaloisKeysNew(galElements, sk)

	newEvaluator := ckks.NewEvaluator(params, rlwe.NewMemEvaluationKeySet(kgen.GenRelinearizationKeyNew(sk), galKeys...))

	return newEvaluator
}

func (obj prevResnetCifar10) myLogSave(fileName string, ctIn *rlwe.Ciphertext) {
	folderName := "myLogs/"
	plainIn := obj.Decryptor.DecryptNew(ctIn)

	floatIn := make([]float64, obj.params.MaxSlots())
	obj.Encoder.Decode(plainIn, floatIn)

	floatToTxt(folderName+fileName+".txt", floatIn)
}

func (obj preBlock) myLogSave(fileName string, ctIn *rlwe.Ciphertext) {
	folderName := "myLogs/"
	plainIn := obj.Decryptor.DecryptNew(ctIn)

	floatIn := make([]float64, obj.params.MaxSlots())
	obj.Encoder.Decode(plainIn, floatIn)

	floatToTxt(folderName+fileName+".txt", floatIn)
}
