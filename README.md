# Rotation Optimized Convolution and Parallel BSGS matrix-vector multiplication.       
This is the supplementary implementation of 'Low-Latency Linear Transformations with Small Key Transmission for Private Neural Network via Homomorphic Encryption.'       

Our implementation is based on [Lattigo v5.0.2](https://github.com/tuneinsight/lattigo/tree/v5.0.2), which is written in Go.

To run this project, please ensure that **Go (version 1.18 or higher)** is installed on your system.

Since we use Lattigo library to run the code, our implementation's location is fixed at FHE_Conv_MatVecMul/examples/rotopt/.    

---

## Run
You can run rotation optimized convolution test function as follows:     
```bash
cd examples/rotopt/   
go run . conv      
```    

Alternatively, you can specify a test function using arguments:   
```bash
go run . parBSGS conv          
```    
---

## Arguments
|Argument|Descript|Related Figure/Table
|------|---|---|
|`basic`|Execution time of rotation, multiplication, addition in our CKKS environment|Fig.1|
|`conv`|Execution time comparison of rotation optimized convolution and multiplexed parallel convolution|Fig.13|
|`otherConv`|Execution time of convolution operations used in convolution-integrated Transformer models and state space models (SSMs): CvT-CIFAR100 Stage 2 & 3, and MUSE_PyramidGenConv|Fig.13|
|`blueprint`|Extract each convolution's blueprint|Appendix A|
|`rotkey`|Hierarchical rotation key system and small level key system test|TABLE 2|
|`fc`|Apply parallel BSGS matrix-vector multiplication to fully connected layer|Fig.14|
|`matVecMul`|Execution time comparison of parallel BSGS matrix-vector multiplication and BSGS diagonal method |Fig.14|
|`paramTest`|Supports various CKKS parameter configurations (`PN16QP1761`, `PN15QP880CI`, `PN16QP1654pq`, `PN15QP827CIpq`) based on [Lattigo's official parameter sets](https://pkg.go.dev/github.com/tuneinsight/lattigo/v4@v4.1.1/ckks#section-readme)|-|
|`ALL`|If you write ALL or don't write any args, all of the test function will be started|-|

---

## Algorithm Structure

The core algorithms are located in the `examples/rotopt/modules` directory.
- `convConfig.go` defines the convolution configurations corresponding to Appendix A and B of the paper.

In addition, implementations for rotation key systems are provided in the examples/rotopt/ directory:
- `hierarchyKey.go` contains the hierarchical rotation key system.
- `smallLevelKey.go` contains the small-level rotation key system.
---


## Notes

- The convolution operations for `otherConv` include:
  - **CvTCifar100Stage2**: Used in Stage 2 of the Convolutional Vision Transformer (CvT) on CIFAR-100.
  - **CvTCifar100Stage3**: Used in Stage 3 of CvT on CIFAR-100.
  - **MUSE_PyramidGenConv**: Used to generate a multi-scale feature pyramid from CLIP in the MUSE model.

- The `paramTest` mode allows evaluating the algorithms under different CKKS parameter configurations.  
  Refer to [Lattigo CKKS documentation](https://pkg.go.dev/github.com/tuneinsight/lattigo/v4@v4.1.1/ckks#section-readme) for more details.

