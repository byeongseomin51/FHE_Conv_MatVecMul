import sys
def afterK(before,K):
    before = before-1
    one_size = 2**(14-(K//2))
    after = ((before%one_size)%(32//K)) * K + (before%one_size)//(32//K)*(32*K)+1
    
    return after%one_size + before//one_size



def returnAfterLocate(kernelNum,k):
    result=[]
    for i in range(kernelNum):
        result.append(i%k+ i%(k*k)//k*32+ i//(k*k)*1024)
    return result
    
    


def getRotIndex():
    # CONV1 2 3s2 3 4 4s2 순
    CONVID= ["CONV1","CONV2","CONV3s2","CONV3","CONV4s2","CONV4"]
    copyNum = [8,2,2,4,4,8]
    kernelNum =[16,16,32,32,64,64]
    afterK = [1,1,2,2,4,4]

    
    for i in range(len(CONVID)):
        print(CONVID[i])
        afterLocate = returnAfterLocate(kernelNum[i],afterK[i])

        results = []
        for eachCipher in range(kernelNum[i]//copyNum[i]):
            for each in range(copyNum[i]):
                beforelocate = each*32768//copyNum[i]
                whichKernel = eachCipher*copyNum[i]+each
                
                rotIndex = beforelocate-afterLocate[whichKernel]
                if rotIndex<0:
                    rotIndex = 32768+rotIndex
                
                results.append(rotIndex)
        print(sorted(results))
        if len(results) != len(set(results)):
            print("!@!!!!!!!")
            print(len(results) - len(set(results)))
        print()
                
                
                
                
    
    

if __name__ =='__main__':
    # input = sys.argv[1]
    # input = int(input)
    # print(input,"->",afterK(input,4))
    getRotIndex()
    
    input = [-33,-32,-31,-1,1,31,32,33]
    for i in range(len(input)):
        if input[i] <0:
            input[i]=32768+input[i]
        
    print(sorted(input))