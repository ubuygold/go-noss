1. 将.env.example改为.env
2. 在.env里配置公私钥。NPUB/NSEC的获取可以使用OKX钱包的nostr链获取地址/导出私钥获取，暂时不知道如何从TP钱包获取NPUB/NSEC。注意npub... nsec..开头的需要转换一下才能使用，可以去https://nostrcheck.me/converter/ 转换
3. 使用golang编译脚本，或者使用release中编译好的版本
4. 运行
5. 再加一句话：判断你是否成功运行脚本：1. 成功运行没闪退 2. 在蹦字 3. 返回值Response显示是200 4. Publish to后面有id，并且id前5位为0
