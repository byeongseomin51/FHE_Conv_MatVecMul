package mulParModules

import (
	// "strconv"

	"github.com/tuneinsight/lattigo/v5/core/rlwe"
	// "github.com/tuneinsight/lattigo/v5/schemes/ckks"
)

type BsgsDiagMatrixMul struct {
	N  int //N=n1*n2
	n1 int
	n2 int
	nt int
}

type ParBsgsDiagMatrixMul struct {
	N  int //N=n1*n2
	n1 int
	n2 int //>=pi, <=nt/(2*N)
	pi int
	nt int
}

func NewBsgsDiagMatrixMul(N int, n1 int, n2 int, nt int) {

}

func NewParBsgsDiagMatrixMul(N int, n1 int, n2 int, nt int) {

}

func (obj BsgsDiagMatrixMul) Foward(ctIn *rlwe.Ciphertext) (ctOut *rlwe.Ciphertext) {
	return ctOut
}

func (obj ParBsgsDiagMatrixMul) Foward(ctIn *rlwe.Ciphertext) (ctOut *rlwe.Ciphertext) {
	return ctOut
}
