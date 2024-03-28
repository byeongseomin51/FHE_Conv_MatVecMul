package mulParModules

import "fmt"

func GetConvMap(convID string, depth int) ([][]int, int, []int) {
	var convMap [][]int
	var q int //length of kernel_map
	var rotIndex3by3Kernel []int

	if convID == "CONV1" { //32*32*3 -> 32*32*16, kernel=3*3, k=1
		//CONV1
		//=================Choose MAP=================//
		//2 depth, 14 rotation
		if depth == 2 {
			convMap = [][]int{ //1499ms
				{4}, //tree length
				{0, 2048},
				{2, 2, 1024},
				{3, 2, 14336},
				{0, -16384},
			}

		} else {
			fmt.Printf("RotOptConv : Invalid parameter! convID(%s), depth(%v)", convID, depth)
		}

		//============================================//
		q = 2
		rotIndex3by3Kernel = []int{-33, -32, -31, -1, 0, 1, 31, 32, 33}

		//========================================================================================//
	} else if convID == "CONV2" { //32*32*16 -> 32*32*16, kernel=3*3, k=1
		//CONV2
		//=================Choose MAP=================//
		if depth == 2 { //5089ms
			//2depth 36 rotations
			convMap = [][]int{
				{3}, //tree length
				{2, 8, 1024, 2048, 4096},
				{3, 4, 8192, 8192, 16384},
				{0, -16384},
			}
		} else if depth == 3 {
			//3 depth, 28 rotation
			convMap = [][]int{ //4224ms
				{4}, //tree length
				{1, 2, 1024},
				{2, 4, 2048, 4096},
				{3, 4, 8192, 8192, 16384},
				{0, -16384},
			}
		} else if depth == 4 {
			//4 depth, 26 rotation
			convMap = [][]int{ //3871ms
				{5}, //tree length
				{1, 2, 1024},
				{1, 2, 2048},
				{2, 2, 4096},
				{3, 4, 8192, 8192, 16384},
				{0, -16384},
			}
		} else {
			fmt.Printf("RotOptConv : Invalid parameter! convID(%s), depth(%v)", convID, depth)
		}

		//============================================//
		q = 8
		rotIndex3by3Kernel = []int{-33, -32, -31, -1, 0, 1, 31, 32, 33}

		//========================================================================================//
	} else if convID == "CONV3s2" { //32*32*16 -> 16*16*32, kernel=3*3, k=1->2
		//CONV3s2
		//=================Choose MAP=================//
		if depth == 2 {
			//2 depth 77 rotation
			convMap = [][]int{ //10854ms
				{4}, //tree length
				{2, 16, 1024, 2048, 4096, 8192},
				{3, 4, 8192 - 1, 16384 - 32, 16384 + 8192 - 32 - 1},
				{0, -8192},
				{0, -16384},
			}
		} else if depth == 3 {
			//3 depth, 53 rotation,
			convMap = [][]int{ //8212ms
				{5}, //tree length
				{1, 4, 1024, 2048},
				{2, 4, 4096, 8192},
				{3, 4, 8192 - 1, 16384 - 32, 16384 + 8192 - 32 - 1},
				{0, -8192},
				{0, -16384},
			}
		} else if depth == 4 {
			//4 depth, 45 rotation,
			convMap = [][]int{ //7688ms
				{6}, //tree length
				{1, 2, 1024},
				{1, 2, 2048},
				{2, 4, 4096, 8192},
				{3, 4, 8192 - 1, 16384 - 32, 16384 + 8192 - 32 - 1},
				{0, -8192},
				{0, -16384},
			}
		} else if depth == 5 {
			//5 depth, 43 rotation,
			convMap = [][]int{ //7688ms
				{7}, //tree length
				{1, 2, 1024},
				{1, 2, 2048},
				{1, 2, 4096},
				{2, 2, 8192},
				{3, 4, 8192 - 1, 16384 - 32, 16384 + 8192 - 32 - 1},
				{0, -8192},
				{0, -16384},
			}
		} else {
			fmt.Printf("RotOptConv : Invalid parameter! convID(%s), depth(%v)", convID, depth)

		}

		//============================================//
		q = 16
		rotIndex3by3Kernel = []int{-33, -32, -31, -1, 0, 1, 31, 32, 33}

		//========================================================================================//
	} else if convID == "CONV3" { //16*16*32 -> 16*16*32, kernel=3*3, k=2
		//CONV3
		//=================Choose MAP=================//
		if depth == 2 {
			//2 depth, 49 rotation
			convMap = [][]int{ //682.29563ms
				{5}, //tree length
				{0, 2048},
				{2, 8, 1, 32, 1024},
				{3, 8, 4096, 4096*2 - 2048, 4096*3 - 2048, 4096*4 - 4096, 4096*5 - 4096, 4096*6 - 6144, 4096*7 - 6144}, // 이렇게 하면9167 ms, 10254 ms {3,16,2048,4096,6144,8192-2048,8192,8192+2048,8192+4096,16384-4096,16384-2048,16384,16384+2048,16384+2048,16384+4096,16384+6144,16384+8192},
				{0, -8192},
				{0, -16384},
			}
			// convMap = [][]int{ //717.244399ms
			// 	{5},                 //tree length
			// 	{0, 2048},           //1
			// 	{0, 4096},           //1
			// 	{2, 8, 1, 32, 1024}, //1
			// 	{3, 4, 8192 - 2048, 16384 - 4096, 16384 + 2048}, //0
			// 	// {3, 8, 4096, 4096*2 - 2048, 4096*3 - 2048, 4096*4 - 4096, 4096*5 - 4096, 4096*6 - 6144, 4096*7 - 6144}, // 이렇게 하면9167 ms, 10254 ms {3,16,2048,4096,6144,8192-2048,8192,8192+2048,8192+4096,16384-4096,16384-2048,16384,16384+2048,16384+2048,16384+4096,16384+6144,16384+8192},
			// 	{0, -8192},  //0
			// 	{0, -16384}, //0
			// }
		} else if depth == 3 {
			//3 depth, 35 rotations
			convMap = [][]int{ //4583 ms
				{7},           //tree length
				{1, 4, 1, 32}, //2
				{0, 2048},     //1
				{0, 4096},     //1
				{2, 2, 1024},  //1
				{3, 4, 8192 - 2048, 16384 - 4096, 16384 + 2048}, //0
				{0, -8192},  //0
				{0, -16384}, //0
			}
		} else if depth == 4 {
			//4 depth, 31 rotation
			convMap = [][]int{ //4247ms
				{8},          //tree length
				{1, 2, 1},    //3
				{1, 2, 32},   //2
				{0, 2048},    //1
				{0, 4096},    //1
				{2, 2, 1024}, //1
				{3, 4, 8192 - 2048, 16384 - 4096, 16384 + 2048}, //0
				{0, -8192},  //0
				{0, -16384}, //0
			}
		} else {
			fmt.Printf("RotOptConv : Invalid parameter! convID(%s), depth(%v)", convID, depth)

		}

		//============================================//
		q = 8
		rotIndex3by3Kernel = []int{-66, -64, -62, -2, 0, 2, 62, 64, 66}

		//========================================================================================//
	} else if convID == "CONV4s2" { //16*16*32 -> 8*8*64, kernel=3*3, k=2->4
		//CONV4s2
		//=================Choose MAP=================//
		if depth == 2 {
			//2 depth, 82 rotation
			convMap = [][]int{ //1.166379268s
				{5}, //tree length
				{2, 16, 1, 32, 1024, 2048},
				{3, 8, 4096, 8192 - 2, 8192 - 2 + 4096, 16384 - 32 - 32, 16384 - 32 - 32 + 4096, 16384 + 8192 - 32 - 32 - 2, 16384 + 8192 - 32 - 32 - 2 + 4096},
				{0, -4096},
				{0, -8192},
				{0, -16384},
			}
			// convMap = [][]int{ //1.280690336s
			// 	{5}, //tree length
			// 	{0, 4096},
			// 	{2, 16, 1, 32, 1024, 2048},
			// 	{3, 4, 8192 - 2, 16384 - 32 - 32, 16384 + 8192 - 32 - 32 - 2},
			// 	{0, -4096},
			// 	{0, -8192},
			// 	{0, -16384},
			// }
		} else if depth == 3 {
			//3 depth, 58 rotation
			convMap = [][]int{ //8492ms
				{7},                //tree length
				{1, 4, 1, 32},      //+2*4 = 8          0 4 8 12
				{0, 4096},          //나중에 없앨 듯 9     0 1 2 3
				{2, 4, 1024, 2048}, //(9+2)*4=44		0 1 2 3
				{3, 4, 8192 - 2, 16384 - 32 - 32, 16384 + 8192 - 32 - 32 - 2}, //47
				{0, -4096},  //48
				{0, -8192},  //49
				{0, -16384}, //50
			}
		} else if depth == 4 {
			//4 depth, 50 rotation
			convMap = [][]int{ //7498ms [4,2,1]
				{8},              //tree length
				{1, 2, 1},        //2    0 2 4 6
				{1, 4, 32, 1024}, //16   0 4
				{0, 4096},        //17   0 1
				{2, 2, 2048},     //36   0 1
				{3, 4, 8192 - 2, 16384 - 32 - 32, 16384 + 8192 - 32 - 32 - 2}, //39
				{0, -4096},
				{0, -8192},
				{0, -16384}, //42
			}
		} else if depth == 5 {
			//5 depth, 46 rotation
			convMap = [][]int{ //7498ms
				{9}, //tree length
				{1, 2, 1},
				{1, 2, 32},
				{1, 2, 1024},
				{0, 4096},
				{2, 2, 2048},
				{3, 4, 8192 - 2, 16384 - 32 - 32, 16384 + 8192 - 32 - 32 - 2},
				{0, -4096},
				{0, -8192},
				{0, -16384},
			}
		} else {
			fmt.Printf("RotOptConv : Invalid parameter! convID(%s), depth(%v)", convID, depth)

		}

		//============================================//
		q = 16
		rotIndex3by3Kernel = []int{-66, -64, -62, -2, 0, 2, 62, 64, 66}

		//========================================================================================//
	} else if convID == "CONV4" { //8*8*64 -> 8*8*64, kernel=3*3, k=4
		//CONV4
		//=================Choose MAP=================//
		if depth == 2 {
			//OPTION 3 : 2 depth, 66 rotation
			convMap = [][]int{ //7906ms
				{8}, //tree length
				{0, 32 + 32},
				{0, 1024},
				{0, 2048},
				{2, 8, 1, 2, 32},
				{3, 8, 1024*4 - 64, 1024 * 7, 1024*11 - 64, 1024 * 14, 1024*18 - 64, 1024 * 21, 1024*25 - 64},
				{0, -4096},
				{0, -8192},
				{0, -16384},
			}
		} else if depth == 3 {
			//3 depth, 42 rotation
			convMap = [][]int{ //5299ms
				{9},          //tree length
				{1, 4, 1, 2}, //2*4=8
				{0, 32 + 32}, //9
				{0, 1024},    //10
				{0, 2048},    //11
				{2, 2, 32},   //(11+1)*2 = 24
				{3, 8, 1024*4 - 64, 1024 * 7, 1024*11 - 64, 1024 * 14, 1024*18 - 64, 1024 * 21, 1024*25 - 64},
				{0, -4096},  //32
				{0, -8192},  //33
				{0, -16384}, //34
			}
		} else if depth == 4 {
			//4 depth, 38 rotation
			convMap = [][]int{ //4809ms
				{10},         //tree length
				{1, 2, 1},    //3
				{1, 2, 2},    //2
				{0, 32 + 32}, //2
				{0, 1024},    //2
				{0, 2048},    //2
				{2, 2, 32},   //1 //아래 0
				{3, 8, 1024*4 - 64, 1024 * 7, 1024*11 - 64, 1024 * 14, 1024*18 - 64, 1024 * 21, 1024*25 - 64},
				{0, -4096},  //0
				{0, -8192},  //0
				{0, -16384}, //0
			}
		} else {
			fmt.Printf("RotOptConv : Invalid parameter! convID(%s), depth(%v)", convID, depth)

		}
		//============================================//

		q = 8
		rotIndex3by3Kernel = []int{-132, -128, -124, -4, 0, 4, 124, 128, 132}
	}
	return convMap, q, rotIndex3by3Kernel
}

func ConvertToConvID(planes int, stride int) string {
	if planes == 3 && stride == 1 {
		return "CONV1"
	} else if planes == 16 && stride == 1 {
		return "CONV2"
	} else if planes == 16 && stride == 2 {
		return "CONV3s2"
	} else if planes == 32 && stride == 1 {
		return "CONV3"
	} else if planes == 32 && stride == 2 {
		return "CONV4s2"
	} else if planes == 64 && stride == 1 {
		return "CONV4"
	}
	return ""
}

func GetConvFeature(convID string) *ConvFeature {
	var result ConvFeature
	// rot -> filter -> add
	if convID == "CONV1" { //32*32*3 -> 32*32*16, kernel=3*3, k=1
		result.Layer = 0
		result.LayerStr = "layer0"
		result.X = 0
		result.Input = 2

		result.InputDataWidth = 32
		result.InputDataHeight = 32
		result.InputDataChannel = 3
		result.KernelSize = 3
		result.KernelNumber = 16
		result.Stride = 1
		result.K = 1
		result.AfterK = 1
		result.BeforeCopy = 8
		result.AfterCopy = 2
		result.q = 2

		result.KernelMap = [][]int{
			{0, 4, 8, 12, 2, 6, 10, 14},
			{1, 5, 9, 13, 3, 7, 11, 15},
		}

	} else if convID == "CONV2" { //32*32*16 -> 32*32*16, kernel=3*3, k=1
		result.Layer = 1
		result.LayerStr = "layer1"
		result.X = 1
		result.Input = 1

		result.InputDataWidth = 32
		result.InputDataHeight = 32
		result.InputDataChannel = 16
		result.KernelSize = 3
		result.KernelNumber = 16
		result.Stride = 1
		result.K = 1
		result.AfterK = 1
		result.BeforeCopy = 2
		result.AfterCopy = 2

		result.q = 8

		result.KernelMap = [][]int{
			{0, 8}, {1, 9}, {2, 10}, {3, 11}, {4, 12}, {5, 13}, {6, 14}, {7, 15},
		}

	} else if convID == "CONV3s2" { //32*32*16 -> 16*16*32, kernel=3*3, k=1->2
		result.Layer = 2
		result.LayerStr = "layer2"
		result.X = 0
		result.Input = 1

		result.InputDataWidth = 32
		result.InputDataHeight = 32
		result.InputDataChannel = 16
		result.KernelSize = 3
		result.KernelNumber = 32
		result.Stride = 2
		result.K = 1
		result.AfterK = 2
		result.BeforeCopy = 2
		result.AfterCopy = 4

		result.KernelMap = [][]int{
			{0, 2}, {4, 6}, {8, 10}, {12, 14}, {16, 18}, {20, 22}, {24, 26}, {28, 30},
			{1, 3}, {5, 7}, {9, 11}, {13, 15}, {17, 19}, {21, 23}, {25, 27}, {29, 31},
		}
		result.q = 16

	} else if convID == "CONV3" { //16*16*32 -> 16*16*32, kernel=3*3, k=2
		result.Layer = 2
		result.LayerStr = "layer2"
		result.X = 2
		result.Input = 2

		result.InputDataWidth = 16
		result.InputDataHeight = 16
		result.InputDataChannel = 32
		result.KernelSize = 3
		result.KernelNumber = 32
		result.Stride = 1
		result.K = 2
		result.AfterK = 2
		result.BeforeCopy = 4
		result.AfterCopy = 4

		result.KernelMap = [][]int{
			{0, 8, 16, 24}, {1, 9, 17, 25}, {2, 10, 18, 26}, {3, 11, 19, 27},
			{4, 12, 20, 28}, {5, 13, 21, 29}, {6, 14, 22, 30}, {7, 15, 23, 31},
		}
		result.q = 8

	} else if convID == "CONV4s2" { //16*16*32 -> 8*8*64, kernel=3*3, k=2->4
		result.Layer = 3
		result.LayerStr = "layer3"
		result.X = 0
		result.Input = 1

		result.InputDataWidth = 16
		result.InputDataHeight = 16
		result.InputDataChannel = 32
		result.KernelSize = 3
		result.KernelNumber = 64
		result.Stride = 2
		result.K = 2
		result.AfterK = 4
		result.BeforeCopy = 4
		result.AfterCopy = 8

		result.KernelMap = [][]int{
			{0, 2, 8, 10}, {1, 3, 9, 11}, {4, 6, 12, 14}, {5, 7, 13, 15},
			{16, 18, 24, 26}, {17, 19, 25, 27}, {20, 22, 28, 30}, {21, 23, 29, 31},
			{32, 34, 40, 42}, {33, 35, 41, 43}, {36, 38, 44, 46}, {37, 39, 45, 47},
			{48, 50, 56, 58}, {49, 51, 57, 59}, {52, 54, 60, 62}, {53, 55, 61, 63},
		}

		result.q = 16

	} else if convID == "CONV4" { //8*8*64 -> 8*8*64, kernel=3*3, k=4
		result.Layer = 3
		result.LayerStr = "layer3"
		result.X = 2
		result.Input = 1

		result.InputDataWidth = 8
		result.InputDataHeight = 8
		result.InputDataChannel = 64
		result.KernelSize = 3
		result.KernelNumber = 64
		result.Stride = 1
		result.K = 4
		result.AfterK = 4
		result.BeforeCopy = 8
		result.AfterCopy = 8

		// result.kernelMap = {
		//     {0,16,32,48,8,24,40,56},{1,17,33,49,9,25,41,57},{2,18,34,50,10,26,42,58},{3,19,35,51,11,27,43,59},
		//     {4,20,36,52,12,28,44,60},{5,21,37,53,13,29,45,61},{6,22,38,54,14,30,46,62},{7,23,39,55,15,31,47,63}
		// };
		result.KernelMap = [][]int{
			{0, 8, 16, 24, 32, 40, 48, 56}, {1, 9, 17, 25, 33, 41, 49, 57}, {2, 10, 18, 26, 34, 42, 50, 58}, {3, 11, 19, 27, 35, 43, 51, 59},
			{4, 12, 20, 28, 36, 44, 52, 60}, {5, 13, 21, 29, 37, 45, 53, 61}, {6, 14, 22, 30, 38, 46, 54, 62}, {7, 15, 23, 31, 39, 47, 55, 63},
		}

		result.q = 8

	}

	return &result
}
func GetMulParConvFeature(convID string) *ConvFeature {
	var result ConvFeature
	// rot -> filter -> add
	if convID == "CONV1" { //32*32*3 -> 32*32*16, kernel=3*3, k=1
		result.Layer = 0
		result.LayerStr = "layer0"
		result.X = 0
		result.Input = 2

		result.InputDataWidth = 32
		result.InputDataHeight = 32
		result.InputDataChannel = 3
		result.KernelSize = 3
		result.KernelNumber = 16
		result.Stride = 1
		result.K = 1
		result.AfterK = 1
		result.BeforeCopy = 8
		result.AfterCopy = 2
		result.q = 2

		result.KernelMap = [][]int{
			{0, 1, 2, 3, 4, 5, 6, 7},
			{8, 9, 10, 11, 12, 13, 14, 15},
		}

	} else if convID == "CONV2" { //32*32*16 -> 32*32*16, kernel=3*3, k=1
		result.Layer = 1
		result.LayerStr = "layer1"
		result.X = 1
		result.Input = 1

		result.InputDataWidth = 32
		result.InputDataHeight = 32
		result.InputDataChannel = 16
		result.KernelSize = 3
		result.KernelNumber = 16
		result.Stride = 1
		result.K = 1
		result.AfterK = 1
		result.BeforeCopy = 2
		result.AfterCopy = 2

		result.q = 8

		result.KernelMap = [][]int{
			{0, 1}, {2, 3}, {4, 5}, {6, 7}, {8, 9}, {10, 11}, {12, 13}, {14, 15},
		}

	} else if convID == "CONV3s2" { //32*32*16 -> 16*16*32, kernel=3*3, k=1->2
		result.Layer = 2
		result.LayerStr = "layer2"
		result.X = 0
		result.Input = 1

		result.InputDataWidth = 32
		result.InputDataHeight = 32
		result.InputDataChannel = 16
		result.KernelSize = 3
		result.KernelNumber = 32
		result.Stride = 2
		result.K = 1
		result.AfterK = 2
		result.BeforeCopy = 2
		result.AfterCopy = 4

		result.KernelMap = [][]int{
			{0, 1}, {2, 3}, {4, 5}, {6, 7}, {8, 9}, {10, 11}, {12, 13}, {14, 15},
			{16, 17}, {18, 19}, {20, 21}, {22, 23}, {24, 25}, {26, 27}, {28, 29}, {30, 31},
		}
		result.q = 16

	} else if convID == "CONV3" { //16*16*32 -> 16*16*32, kernel=3*3, k=2
		result.Layer = 2
		result.LayerStr = "layer2"
		result.X = 2
		result.Input = 2

		result.InputDataWidth = 16
		result.InputDataHeight = 16
		result.InputDataChannel = 32
		result.KernelSize = 3
		result.KernelNumber = 32
		result.Stride = 1
		result.K = 2
		result.AfterK = 2
		result.BeforeCopy = 4
		result.AfterCopy = 4

		result.KernelMap = [][]int{
			{0, 1, 2, 3}, {4, 5, 6, 7}, {8, 9, 10, 11}, {12, 13, 14, 15},
			{16, 17, 18, 19}, {20, 21, 22, 23}, {24, 25, 26, 27}, {28, 29, 30, 31},
		}
		result.q = 8

	} else if convID == "CONV4s2" { //16*16*32 -> 8*8*64, kernel=3*3, k=2->4
		result.Layer = 3
		result.LayerStr = "layer3"
		result.X = 0
		result.Input = 1

		result.InputDataWidth = 16
		result.InputDataHeight = 16
		result.InputDataChannel = 32
		result.KernelSize = 3
		result.KernelNumber = 64
		result.Stride = 2
		result.K = 2
		result.AfterK = 4
		result.BeforeCopy = 4
		result.AfterCopy = 8

		result.KernelMap = [][]int{
			{0, 1, 2, 3}, {4, 5, 6, 7}, {8, 9, 10, 11}, {12, 13, 14, 15},
			{16, 17, 18, 19}, {20, 21, 22, 23}, {24, 25, 26, 27}, {28, 29, 30, 31},
			{32, 33, 34, 35}, {36, 37, 38, 39}, {40, 41, 42, 43}, {44, 45, 46, 47},
			{48, 49, 50, 51}, {52, 53, 54, 55}, {56, 57, 58, 59}, {60, 61, 62, 63},
		}

		result.q = 16

	} else if convID == "CONV4" { //8*8*64 -> 8*8*64, kernel=3*3, k=4
		result.Layer = 3
		result.LayerStr = "layer3"
		result.X = 2
		result.Input = 1

		result.InputDataWidth = 8
		result.InputDataHeight = 8
		result.InputDataChannel = 64
		result.KernelSize = 3
		result.KernelNumber = 64
		result.Stride = 1
		result.K = 4
		result.AfterK = 4
		result.BeforeCopy = 8
		result.AfterCopy = 8

		// result.kernelMap = {
		//     {0,16,32,48,8,24,40,56},{1,17,33,49,9,25,41,57},{2,18,34,50,10,26,42,58},{3,19,35,51,11,27,43,59},
		//     {4,20,36,52,12,28,44,60},{5,21,37,53,13,29,45,61},{6,22,38,54,14,30,46,62},{7,23,39,55,15,31,47,63}
		// };
		result.KernelMap = [][]int{
			{0, 1, 2, 3, 4, 5, 6, 7}, {8, 9, 10, 11, 12, 13, 14, 15}, {16, 17, 18, 19, 20, 21, 22, 23}, {24, 25, 26, 27, 28, 29, 30, 31},
			{32, 33, 34, 35, 36, 37, 38, 39}, {40, 41, 42, 43, 44, 45, 46, 47}, {48, 49, 50, 51, 52, 53, 54, 55}, {56, 57, 58, 59, 60, 61, 62, 63},
		}

		result.q = 8

	}

	return &result
}

type ConvFeature struct {
	Layer            int
	LayerStr         string
	X                int
	Input            int
	InputDataWidth   int
	InputDataHeight  int
	InputDataChannel int
	KernelSize       int
	KernelNumber     int
	Stride           int
	K                int
	AfterK           int
	BeforeCopy       int
	AfterCopy        int
	KernelMap        [][]int
	Split            int
	q                int
}
