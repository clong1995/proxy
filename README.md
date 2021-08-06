# proxy tcp代理程序
## 原生golang开发，无第三方依赖。可实现公网访问内网，或网络中电脑的消息传递。
在内网电脑上执行:
``` bash
mian -server 1.2.3.4:56 -client 127.0.0.1:8080
```
在公网服务器上执行:
``` bash
mian -server :56 -public :8080
```  
## 链路实现  
