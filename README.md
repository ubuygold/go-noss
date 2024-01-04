1. 将.env.example改为.env
2. 在.env里配置公私钥。NPUB/NSEC的获取可以使用OKX钱包的nostr链获取地址/导出私钥获取，暂时不知道如何从TP钱包获取NPUB/NSEC。注意npub... nsec..开头的需要转换一下才能使用，可以去https://nostrcheck.me/converter/ 转换
3. 使用golang编译脚本，或者使用release中编译好的版本
4. 运行
