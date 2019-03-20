Payment:

1. 业务背景:
用户A购买用户B的服务, A需要付钱(转账)给B.
A的账户是银行org1的.
B的账户是银行org2的.
所以本质上是个跨行清算的问题. 清算业务的传统做法效率低(慢),成本高.

2. 使用fabric的原生解决方案:
org1,2,3组成一个联盟链, 并在同一个channel上, 共同记账. 即,同一个账本. 一式三份,存放在3个org自己的peer上.
部署并初始化转账的智能合约(ChainCode, cc)的时候, 将该cc的policy定义为 org1 AND org2, 也就是说该cc需要被org1和org2签名认可才行.
账本上每个org初始化状态为:
org1, 100
org2, 100
org3, 100
此处100既为初始化余额,类似于jpm coin, 当然也可以不要, 视具体情况而定.


当A需要给B转账10块钱的时候,
org1的内部系统会做一系列操作, 其中包括 org1.client 触发一笔区块链系统的transaction transfer(org1, org2,10)
cc会做一系列判断然后更新数据库的操作:
- tx的发起者是否是org1
- org1的余额是否充足
- 交易额是否大于0
- 是否是自己给自己转账
...
- 更新状态数据库. (org1, 90), (org2, 110)

最终这些交易记录还会放到区块链上.包括org3.
可以使用 fabric 提供的 private data collection 功能, 将org1,2之间的交易数据只存储在org1,2的 private database 里, 区块链上只放tx的哈希.
这样org3就无法看到org1和org2的交易具体信息,但是又具有见证人的功能.

3. 使用fabric的原生解决方案的问题:
虽然org1和org2的交易数据放在自己的private database里, 但是由于不在TEE的环境里(threatened by meltdown, specter, etc.), 这些数据任然会泄露, 很不安全.

4. Fabric + TEE
所有存放到 private database里的数据都使用 AES 加密. 
把cc的执行搬到TEE环境中去(ASCE, Accelor Smart Contract Engine), 
    - feed给cc的confidential数据(来自org1.client或者private database)都是加密过的.  在CC执行前, 使用FPGA中的datakey进行解密.
    - cc的执行结果中的confidential数据会被FPGA中的datakey进行加密后返回给fabric peer.

Notes:
因为区块链上存放的是加密后的交易的哈希, 所以历史交易记录需要org自己额外维护.
如果datakey更改过,那么交易记录加密所使用的datakey也要org自己额外维护.

-----------------------------
Auction:

1. 业务背景:
org1发起一个auction 并设定了规则(价高者得)以及有效时间.
org2和org3参与竞拍,在给定的时间内各自出价.
有效时间到了之后, org1 结束auction, 宣布获胜买家.

2. 使用fabric的原生解决方案:
org1(拍卖行), org2 & org3(bidder) 组成一个联盟链, 并在同一个channel上, 共同记账. 即,同一个账本. 一式三份,存放在3个org自己的peer上.
Auction智能合约(ChainCode, cc)只部署在org1的peer上, 并且该cc的policy定义为 org1, 也就是说该cc只需要org1的背书即可.
org1 单独使用一个 private data collection, 用来记录来自org2和3的bid, 以及在auction结束后算出 winner.
因为private data collection 的tx的哈希会存在org2,3的peer上, 所以org2,3看不到对方bid的具体信息,但是可以见证tx.

org1.client 发起一个创建auction的交易, cc执行并在 private state db 上新建1个记录: pair(key, value):
key := auction1,
value := {
    winner = empty
    price = 0
    start = current time
    end = current time + 1 hour
}

org2.client在限定的时间内(多次)参与竞拍, cc执行并在 private data db 上新建/修改 1个记录: pair(key, value):
{
    key := auction1.org2
    value := 200
}

org3.client在限定的时间内(多次)参与竞拍, cc执行并在 private data db 上新建/修改 1个记录: pair(key, value):
{
    key := auction1.org3
    value := 300
}

竞拍截止时, org1发起EndAuction的tx, cc会去比较 auction1.org2 和 auction1.org3 的value, 取其大者, 然后更新 auction1.state.winner 和 auction1.state.price.

3. 使用fabric的原生解决方案的问题:
 -1. 虽然org2和org3的出价放在org1的private database里, org2和org3都无法得知对方的出价, 但是由于不在TEE的环境里(threatened by meltdown, specter, etc.), 这些数据仍然会泄露, 很不安全.
 -2. org2和org3无法发现org1是否作弊, 解决办法是:
     org2,3将自己的bid加密. 在auction结束后, 把解密的秘钥发给org1.
     org1发起EndAuction的tx, cc便可算出winner bidder.
     但是这样对于org2,3来说太麻烦了, 所以暂不考虑(也可以使用同态加密).

4. Fabric + TEE
所有存放到 private database里的数据都使用 AES 加密.
把cc的执行搬到TEE环境中去(ASCE, Accelor Smart Contract Engine),
    - feed给cc的confidential数据(来自org1.client或者private database)都是加密过的.  在CC执行前, 使用FPGA中的datakey进行解密.
    - cc的执行结果中的confidential数据会被FPGA中的datakey进行加密后返回给fabric peer.


Notes:
因为区块链上存放的是加密后的交易的哈希, 所以历史交易记录需要org自己额外维护.
如果datakey更改过,那么交易记录加密所使用的datakey也要org自己额外维护.







