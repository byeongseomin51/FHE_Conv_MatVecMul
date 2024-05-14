package main

import (
	"fmt"

	"github.com/tuneinsight/lattigo/v5/core/rlwe"
	"github.com/tuneinsight/lattigo/v5/schemes/ckks"
)

// ///////////////////////////////////////////////////////////////////////////////////////////////////////
// Small level key system
// ///////////////////////////////////////////////////////////////////////////////////////////////////////
type SmallLevelKey struct {
	RotIndex     int
	MaxMultLevel int
	KeyLevel     int
	N            int
	K            int
	Dnum         int
	Hdnum        float64
	LogP         float64
	LogQ         float64
	LogPQ        float64

	galKey *rlwe.GaloisKey
}

func NewSmallLevelKey(rotIndex int, keyLevel int, maxMultLevel int, params *ckks.Parameters) *SmallLevelKey {

	sm := len(params.LogPi())

	split_ind := []int{}
	split_len := []int{}
	split_ind = append(split_ind, 0)
	logq := params.LogQi()
	sum_P := 0
	for _, val := range params.LogPi() {
		sum_P += val
	}
	sum := 0
	for i := 0; i < len(logq); i++ {
		sum += logq[i]
		if sum > sum_P {
			split_len = append(split_len, i-split_ind[len(split_ind)-1])
			split_ind = append(split_ind, i)
			sum = logq[i]
		}

	}
	dnum := len(split_ind)

	logp := params.LogP()

	slicedLogQ := 0
	for i := 0; i < maxMultLevel; i++ {
		slicedLogQ += logq[i]
	}

	hdnum := float64(slicedLogQ) / logp
	return &SmallLevelKey{
		RotIndex:     rotIndex,
		MaxMultLevel: maxMultLevel,
		KeyLevel:     keyLevel,
		Dnum:         dnum,
		Hdnum:        hdnum,

		N: params.N(),
		K: sm,

		LogP:  logp,
		LogQ:  float64(slicedLogQ),
		LogPQ: float64(slicedLogQ) + logp,
	}
}

func (obj SmallLevelKey) GetKeySize() int {
	uint64Size := 8 //byte
	N := obj.N
	L := obj.MaxMultLevel
	K := obj.K 
	dnum := obj.Dnum
	keySize := uint64Size * N * (L + 1 + K) * 2 * dnum

	return keySize
}

func (obj SmallLevelKey) PrintKeyInfo() {
	fmt.Println("MaxMultLevel : ", obj.MaxMultLevel)
	fmt.Println("KeyLevel : ", obj.KeyLevel)
	fmt.Println("N : ", obj.N)
	fmt.Println("K : ", obj.K)
	fmt.Println("Dnum : ", obj.Dnum)
	fmt.Println("Hdnum : ", obj.Hdnum)
	fmt.Println("LogP : ", obj.LogP)
	fmt.Println("LogQ : ", obj.LogQ)
	fmt.Println("LogPQ : ", obj.LogPQ)
}

func GenLevelUpKey(preLevelKey *SmallLevelKey, newHdnum float64) *SmallLevelKey {
	logPQbound := 1792

	newLogQ := preLevelKey.LogPQ
	newLogP := newLogQ / newHdnum

	newLogQP := newLogQ + newLogP

	// fmt.Println("During Key Level Up! newHdnum = ", newHdnum, " , newLogQP = ", newLogQP)
	if newLogQP >= float64(logPQbound) {
		fmt.Println("Error in GenLevelUpKey!!! : logPQ bound = ", newLogQP, ">=1792")
	}

	return &SmallLevelKey{
		RotIndex:     preLevelKey.RotIndex,
		MaxMultLevel: preLevelKey.MaxMultLevel,
		KeyLevel:     preLevelKey.KeyLevel + 1,
		Dnum:         preLevelKey.Dnum,
		Hdnum:        newHdnum,

		N: preLevelKey.N,
		K: int(newLogP) / 60,

		LogP:  newLogP,
		LogQ:  newLogQ,
		LogPQ: newLogQ + newLogP,
	}
}
