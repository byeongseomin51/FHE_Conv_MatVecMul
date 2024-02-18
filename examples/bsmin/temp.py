import sys
def afterK(before,K):
    before = before-1
    one_size = 2**(14-(K//2))
    after = ((before%one_size)%(32//K)) * K + (before%one_size)//(32//K)*(32*K)+1
    
    return after%one_size + before//one_size

    
    

if __name__ =='__main__':
    input = sys.argv[1]
    input = int(input)
    print(input,"->",afterK(input,4))