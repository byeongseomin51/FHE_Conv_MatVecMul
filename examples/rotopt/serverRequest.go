package main

import (
	"fmt"
	"rotopt/core"

	"github.com/tuneinsight/lattigo/v5/core/rlwe"
	"github.com/tuneinsight/lattigo/v5/schemes/ckks"
)

func rotIndexToGaloisEl(input []int, params ckks.Parameters, kgen *rlwe.KeyGenerator, sk *rlwe.SecretKey) *ckks.Evaluator {
	var galElements []uint64

	for _, rotIndex := range input {
		galElements = append(galElements, params.GaloisElement(rotIndex))
	}
	galKeys := kgen.GenGaloisKeysNew(galElements, sk)

	for i := 0; i < len(galKeys); i++ {
		// fmt.Println(unsafe.Sizeof(*galKeys[0]), unsafe.Sizeof(galKeys[0].GaloisElement), unsafe.Sizeof(galKeys[0].NthRoot), unsafe.Sizeof(galKeys[0].EvaluationKey), unsafe.Sizeof(galKeys[0].GadgetCiphertext), unsafe.Sizeof(galKeys[0].BaseTwoDecomposition), unsafe.Sizeof(galKeys[0].Value))
		//일단 48 byte 인듯
		// fmt.Println(galKeys[i].LevelP(), galKeys[i].LevelQ())
	}
	newEvaluator := ckks.NewEvaluator(params, rlwe.NewMemEvaluationKeySet(kgen.GenRelinearizationKeyNew(sk), galKeys...))

	return newEvaluator
}

// Use Level0 keys of resnet, this func return what kinds of level1 rot key is needed.
// And make graph by using Level0RotKeyNeeded and Level1Rot keys
func Level1RotKeyNeededForInference(Level0RotKeyNeeded []int) []int {

	//Find which level1 key is needed...
	var level1 []int

	//Max 16384
	step := 4
	stepCount := 7

	//Max 4096
	// step := 16
	// stepCount := 3

	//max 1024
	// step := 32
	// stepCount := 2

	rotIndex := 1
	for i := -1; i < stepCount; i++ {
		level1 = append(level1, rotIndex)
		level1 = append(level1, -rotIndex)
		rotIndex *= step
	}

	level0 := Level0RotKeyNeeded
	fmt.Println("Required Level 0 : ")
	fmt.Println(level0)

	//Make graph with this.
	// nodes, graph, Hgraph := MakeGraph(level0, level1)
	_, graph, _ := MakeGraph(level0, level1)
	fmt.Println("Graph created!")

	//Make MST
	parent := PrimMST(graph)
	fmt.Println("MST created!")

	// Find minimum path.
	// for targetNode := 1; targetNode < len(nodes); targetNode++ {
	// 	minPath := findPath(0, targetNode, parent)
	// 	fmt.Print("Mimum path to ", targetNode, ":", minPath, " ")
	// 	for start := 1; start < len(minPath); start++ {
	// 		fmt.Print(nodes[minPath[start-1]].eachInt, Hgraph[minPath[start-1]][minPath[start]], nodes[minPath[start]].eachInt, "->")
	// 	}
	// 	fmt.Println()

	// }
	//Print MST sum and average
	mstSum := 0
	for i := 1; i < len(graph); i++ {
		mstSum += graph[i][parent[i]]
	}
	fmt.Println("MST sum and average : ", mstSum, float64(mstSum)/float64(len(parent)))

	return level1

}

// Return level0 needed rotation keys for mulpar and rotopt
func RotKeyOrganize(layer int) ([]int, []int) {
	// register
	convIDs := []string{"CONV1", "CONV2", "CONV3s2", "CONV3", "CONV4s2", "CONV4"}
	maxDepth := []int{2, 2, 2, 2, 2, 2}
	maxDepthVal := 2
	rotOptRot := make([][]int, maxDepthVal+1)

	// Get RotOptConv all rotation index
	for i := 0; i < len(convIDs); i++ {
		rots := core.RotOptConvRegister(convIDs[i], maxDepth[i])
		for level := 0; level < maxDepthVal+1; level++ {
			for _, each := range rots[level] {
				rotOptRot[level] = append(rotOptRot[level], each)
			}
		}
	}
	rotOptRot = OrganizeRot(rotOptRot)

	// Print all rot index
	length := 0
	for _, i := range rotOptRot {
		length += len(i)
		// fmt.Println(len(i))
	}
	fmt.Println("Rotation Optimized Convolution total required key-level 0 rotation key number :", length)
	fmt.Println(rotOptRot)

	//Linearize Rot Keys
	var resultRotOpt []int
	for _, i := range rotOptRot {
		for _, each := range i {
			resultRotOpt = append(resultRotOpt, each)
		}
	}
	//remove duplicate
	resultRotOpt = removeDuplicates(resultRotOpt)

	// Get MulParConv all rotation index
	mulParRot := make([][]int, 3)
	for i := 0; i < len(convIDs); i++ {
		rots := core.MulParConvRegister(convIDs[i])
		for level := 0; level < maxDepthVal+1; level++ {
			for _, each := range rots[level] {
				mulParRot[level] = append(mulParRot[level], each)
			}
		}
	}
	mulParRot = OrganizeRot(mulParRot)

	length = 0
	for _, i := range mulParRot {
		length += len(i)
		// fmt.Println(len(i))
	}
	fmt.Println("Multiplexed Parallel Convolution total required key-level 0 rotation key number :", length)
	fmt.Println(mulParRot)

	//Linearize Rot Keys
	var resultMulPar []int
	for _, i := range mulParRot {
		for _, each := range i {
			resultMulPar = append(resultMulPar, each)
		}
	}
	//remove duplicate
	resultMulPar = removeDuplicates(resultMulPar)

	// fmt.Println("Remove duplicate then ", len(result))

	return resultMulPar, resultRotOpt
}

func removeDuplicates(nums []int) []int {
	encountered := map[int]bool{}
	result := []int{}

	for v := range nums {
		if encountered[nums[v]] != true {
			encountered[nums[v]] = true
			result = append(result, nums[v])
		}
	}
	return result
}
