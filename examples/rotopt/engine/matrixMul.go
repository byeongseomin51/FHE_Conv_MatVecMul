package engine

import (
	// "strconv"

	"fmt"
	"math"

	"github.com/tuneinsight/lattigo/v5/core/rlwe"
	"github.com/tuneinsight/lattigo/v5/schemes/ckks"
	// "github.com/tuneinsight/lattigo/v5/schemes/ckks"
)

// ///////////////////////////////////////////////////////////////////////////////////////////////////////
// Conventional BSGS diagonal method and proposed Parallel BSGS matrix-vector multiplication.
// ///////////////////////////////////////////////////////////////////////////////////////////////////////
type BsgsDiagMatVecMul struct {
	Evaluator *ckks.Evaluator
	N         int //N=n1*n2
	n1        int
	n2        int
	nt        int
	d         []*rlwe.Plaintext
}

type ParBsgsDiagMatVecMul struct {
	Evaluator *ckks.Evaluator
	N         int //N=n1*n2
	n1        int
	n2        int //>=pi, <=nt/(2*N)
	pi        int
	nt        int
	D         []*rlwe.Plaintext
}

func NewBsgsDiagMatVecMul(weight [][]float64, N int, nt int, ev *ckks.Evaluator, ec *ckks.Encoder, params ckks.Parameters) *BsgsDiagMatVecMul { //weight -> N*N matrix
	if len(weight) != N || len(weight[0]) != N {
		fmt.Println("Wrong size!")
	}

	//make n1, n2
	n1, n2 := FindBsgsSol(N)
	fmt.Println("Automatic setting : n1=", n1, ", n2=", n2)

	//make d
	plaind := make([]*rlwe.Plaintext, N)
	d := diagonalized(weight, N, nt)
	for i := 0; i < n2; i++ {
		for j := 0; j < n1; j++ {
			d[i*n1+j] = rotate(d[i*n1+j], -n1*i)
		}
	}
	for i := 0; i < len(d); i++ {
		plaind[i] = floatToPlain(d[i], ec, params)
	}

	return &BsgsDiagMatVecMul{
		Evaluator: ev,
		N:         N,
		n1:        n1,
		n2:        n2,
		nt:        nt,
		d:         plaind,
	}
}

func NewParBsgsDiagMatVecMul(weight [][]float64, N int, nt int, pi int, ev *ckks.Evaluator, ec *ckks.Encoder, params ckks.Parameters) *ParBsgsDiagMatVecMul {
	if len(weight) != N || len(weight[0]) != N {
		fmt.Println("Wrong size!")
	}
	//make n1, n2
	n1, n2 := FindParBsgsSol(N, nt, pi)
	fmt.Println("Automatic setting :n1=", n1, ", n2=", n2)

	//make D
	d := diagonalized(weight, N, nt)
	D := make([][]float64, n1)
	for i := 0; i < n1; i++ {
		D[i] = make([]float64, nt)
		for j := 0; j < n2; j++ {
			tempD := rotate(d[j*n1+i], -n1*j-(nt/n2)*j)
			D[i] = add(D[i], tempD)
		}
	}

	plainD := make([]*rlwe.Plaintext, n1)
	for i := 0; i < n1; i++ {
		plainD[i] = floatToPlain(D[i], ec, params)
	}

	return &ParBsgsDiagMatVecMul{
		Evaluator: ev,
		N:         N,
		n1:        n1,
		n2:        n2,
		nt:        nt,
		pi:        pi,
		D:         plainD,
	}
}

func (obj BsgsDiagMatVecMul) Foward(ctIn *rlwe.Ciphertext) (ctOut *rlwe.Ciphertext) {
	temp, err := obj.Evaluator.RotateNew(ctIn, -obj.N)
	ErrorPrint(err)
	obj.Evaluator.Add(ctIn, temp, ctIn)

	preRotatedB := make([]*rlwe.Ciphertext, obj.n1)
	preRotatedB[0] = ctIn
	for i := 1; i < obj.n1; i++ {
		preRotatedB[i], err = obj.Evaluator.RotateNew(ctIn, i)
		ErrorPrint(err)
	}

	for i := 0; i < obj.n2; i++ {
		if i == 0 {
			for j := 0; j < obj.n1; j++ {
				if j == 0 {
					ctOut, err = obj.Evaluator.MulNew(preRotatedB[j], obj.d[i*obj.n1+j])
					ErrorPrint(err)
				} else {
					tempB, err := obj.Evaluator.MulNew(preRotatedB[j], obj.d[i*obj.n1+j])
					ErrorPrint(err)
					err = obj.Evaluator.Add(ctOut, tempB, ctOut)
					ErrorPrint(err)
				}
			}
			err = obj.Evaluator.Rescale(ctOut, ctOut)
			ErrorPrint(err)
		} else {
			var tempAnswer *rlwe.Ciphertext
			for j := 0; j < obj.n1; j++ {
				if j == 0 {
					tempAnswer, err = obj.Evaluator.MulNew(preRotatedB[j], obj.d[i*obj.n1+j])
					ErrorPrint(err)
				} else {
					tempB, err := obj.Evaluator.MulNew(preRotatedB[j], obj.d[i*obj.n1+j])
					ErrorPrint(err)
					err = obj.Evaluator.Add(tempAnswer, tempB, tempAnswer)
					ErrorPrint(err)
				}
			}
			err = obj.Evaluator.Rescale(tempAnswer, tempAnswer)
			ErrorPrint(err)
			tempAnswerRotated, err := obj.Evaluator.RotateNew(tempAnswer, obj.n1*i)
			ErrorPrint(err)
			err = obj.Evaluator.Add(ctOut, tempAnswerRotated, ctOut)
			ErrorPrint(err)
		}
	}

	return ctOut
}

func (obj ParBsgsDiagMatVecMul) Foward(ctIn *rlwe.Ciphertext) (ctOut *rlwe.Ciphertext) {
	//for non-fit size ciphertext
	temp, err := obj.Evaluator.RotateNew(ctIn, -obj.N)
	ErrorPrint(err)
	obj.Evaluator.Add(ctIn, temp, ctIn)
	//make it more parallel if needed
	for i := 2; i <= obj.n2/obj.pi; i *= 2 {
		tempB, err := obj.Evaluator.RotateNew(ctIn, -(obj.nt/obj.pi)/i)
		ErrorPrint(err)
		err = obj.Evaluator.Add(ctIn, tempB, ctIn)
		ErrorPrint(err)
	}
	//multiplication
	for i := 0; i < obj.n1; i++ {
		if i == 0 {
			ctOut, err = obj.Evaluator.MulNew(ctIn, obj.D[i])
			ErrorPrint(err)
		} else {
			rotatedB, err := obj.Evaluator.RotateNew(ctIn, i)
			ErrorPrint(err)
			tempB, err := obj.Evaluator.MulNew(rotatedB, obj.D[i])
			ErrorPrint(err)
			err = obj.Evaluator.Add(tempB, ctOut, ctOut)
			ErrorPrint(err)
		}
	}
	err = obj.Evaluator.Rescale(ctOut, ctOut)
	ErrorPrint(err)

	//gather result
	for i := 1; i < obj.n2; i *= 2 {
		tempAnswer, err := obj.Evaluator.RotateNew(ctOut, obj.n1*i+(obj.nt/obj.n2)*i)
		ErrorPrint(err)
		err = obj.Evaluator.Add(ctOut, tempAnswer, ctOut)
		ErrorPrint(err)
	}

	return ctOut
}

func diagonalized(weight [][]float64, N int, nt int) [][]float64 {
	d := make([][]float64, N)

	for i := 0; i < N; i++ {
		d[i] = make([]float64, nt)
		for j := 0; j < N; j++ {
			index := j + i
			if index >= N {
				index = index % N
			}
			d[i][j] = weight[j][index]
		}
	}
	return d
}

func FindBsgsSol(N int) (min_n1 int, min_n2 int) {
	min_sum := 999999999999

	for n1 := 1; n1 < int(math.Sqrt(float64(N)))+1; n1++ {
		if N%n1 == 0 {
			n2 := N / n1
			current_sum := n1 + n2
			if current_sum < min_sum {
				min_sum = current_sum
				min_n1 = n1
				min_n2 = n2
			}
		}
	}
	return min_n1, min_n2
}
func FindParBsgsSol(N int, nt int, pi int) (min_n1 int, min_n2 int) {
	rot_num := 999999999999
	mul_num := 999999999999
	// add_num := 999999999999

	for n2 := pi; n2 <= nt/(2*N); n2++ {
		if N%n2 == 0 {
			n1 := N / n2

			cur_rot := 2*int(math.Log2(float64(n2))) + n1 - int(math.Log2(float64(pi)))
			cur_mul := n1
			// cur_add := n1 + int(math.Log2(float64(n2))) + 1

			if cur_rot < rot_num {
				rot_num = cur_rot
				mul_num = cur_mul
				// add_num = cur_add
				min_n1 = n1
				min_n2 = n2
			} else if cur_rot == rot_num {
				if cur_mul < mul_num {
					rot_num = cur_rot
					mul_num = cur_mul
					// add_num = cur_add
					min_n1 = n1
					min_n2 = n2
				}
			}
		}
	}
	// fmt.Println(N, min_n1, min_n2)
	return min_n1, min_n2
}

func BsgsDiagMatVecMulRegister(N int) []int {
	n1, n2 := FindBsgsSol(N)
	result := make([]int, 0)
	for i := 1; i < n1; i++ {
		result = append(result, i)
	}
	for i := 0; i < n2; i++ {
		result = append(result, n1*i)
	}
	result = append(result, -N)
	return result
}

func ParBsgsDiagMatVecMulRegister(N int, nt int, pi int) []int {
	n1, n2 := FindParBsgsSol(N, nt, pi)
	result := make([]int, 0)
	for i := 2; i <= n2/pi; i *= 2 {
		result = append(result, -(nt/pi)/i)
	}
	for i := 0; i < n1; i++ {
		result = append(result, i)
	}
	for i := 1; i < n2; i *= 2 {
		result = append(result, n1*i+(nt/n2)*i)
	}
	result = append(result, -N)
	return result
}
