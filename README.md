# Rotation Optimized Convolution and Parallel BSGS matrix-vector multiplication.       
This is the supplementary implementation of 'Low-Latency Linear Transformations with Small Key Transmission for Private Neural Network on Homomorphic Encryption.'       

Our implementation is based on lattigo v5.0.2, which is written in Go.             
https://github.com/tuneinsight/lattigo/tree/v5.0.2

To run this project, please ensure that Go (version 1.18 or higher) is installed on your system.

Since we use Lattigo library to run the code, our implementation's location is fixed at FHE_Conv_MatVecMul/examples/rotopt/.    

## run
You can run rotation optimized convolution test function as below.     
```   
cd examples/rotopt/   
go run . conv      
```    

Or you can choose some test function by arguments as below.     
```
go run . parBSGS conv          
```

These are our arguments option. 

|args|descript|image
|------|---|---|
|basic|Execution time of rotation, multiplication, addition in our CKKS environment|Fig.1|
|conv|Execution time comparison of rotation optimized convolution and multiplexed parallel convolution|Fig.13|
|blueprint|Extract current convolution's blueprint|Appendix A|
|downsamp|Execution time comparison of rotation optimized downsampling and multiplexed parallel downsampling|Fig.14|
|rotkey|Hierarchical rotation key system and small level key system test|TABLE 2|
|fc|Apply parallel BSGS matrix-vector multiplication to fully connected layer.|Fig.15|
|parBSGS|Execution time comparison of parallel BSGS matrix-vector multiplication and BSGS diagonal method. |Fig.15|
|ALL|If you write ALL or don't write any args, all of the test function will be started.||

## Algorithm    
All of our main algorithms are implemented in examples/rotopt/engine directory.      
Especially, convConfig.go correspons to that of APPENDIX A and APPENDIX B.       
(Instead of Hierarchical rotation key system or small level key system, which implemented in hierarchyKey.go and smallLevelKey.go).       
