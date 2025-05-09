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
|fc|Apply parallel BSGS matrix-vector multiplication to fully connected layer|Fig.15|
|matVecMul|Execution time comparison of parallel BSGS matrix-vector multiplication and BSGS diagonal method |Fig.15|
|ALL|If you write ALL or don't write any args, all of the test function will be started||

## Algorithm    
All of our main algorithms are implemented in examples/rotopt/modules directory.      
Especially, convConfig.go correspons to that of APPENDIX A and APPENDIX B.       
(Instead of Hierarchical rotation key system or small level key system, which implemented in hierarchyKey.go and smallLevelKey.go).       


## Additional Experiments
New argument options have been added to demonstrate the generality and applicability of our algorithms.

For `otherConv` option, we implemented three types of convolution operations, used in the [Convolutional Vision Transformer (CvT)](https://openaccess.thecvf.com/content/ICCV2021/html/Wu_CvT_Introducing_Convolutions_to_Vision_Transformers_ICCV_2021_paper.html) and the Mamba-based text-video retrieval model ([MUSE: Mamba is an Efficient Multi-scale Learner for Text-Video Retrieval](https://ojs.aaai.org/index.php/AAAI/article/view/32778)).     
Each convolution detail:
- **CvTCifar100Stage2**: This convolution embedding is used in Stage 2 of the Convolutional Vision Transformer (CvT) when performing inference on the CIFAR-100 dataset.
- **CvTCifar100Stage3**: This convolution embedding is used in Stage 3 of CvT for the CIFAR-100 dataset.
- **MUSE_PyramidGenConv**: This convolution is used to generate a multi-scale feature pyramid from the final single-scale feature map of CLIP in the MUSE model.
- 
For `paramTest` option, we support various CKKS parameter configurations, including:     

- `PN16QP1761`
- `PN15QP880CI`
- `PN16QP1654pq`
- `PN15QP827CIpq`

These configurations are described in the [Lattigo official documentation](https://pkg.go.dev/github.com/tuneinsight/lattigo/v4@v4.1.1/ckks#section-readme).
Please refers to the documentation for more details about CKKS parameter configurations.    

|Argument|Description|
|------|---|
|otherConv|Implementation of convolution operations used in convolution-integrated Transformer architectures and state space models (SSMs)|
|paramTest|All algorithms can be tested under various CKKS parameter configurations|

The results for otherConv are as follows:

<img src="https://github.com/user-attachments/assets/4f9cf178-aff8-4e9e-879c-cbb2129b70e5" width="300"/>
<img src="https://github.com/user-attachments/assets/b4e824ff-81be-4bb7-bf2d-6104197cfafd" width="300"/>
<img src="https://github.com/user-attachments/assets/4266140e-3e2b-48f7-8cd7-f798e9161e6c" width="300"/>


