# go_ddns
自用的go语言编写的aliyun记录ipv6服务端
## 功能说明
1. get / 时显示来访IP
2. get /list 时显示密码提交html内容
3. post /list 时检查密码，密码正确列出缓存的有效值，按at时间戳从大往小排序，其中at的时间戳以东8区格式化显示；密码错误则显示来访IP
4. post / 时为提交数据参数有host和ipv6,保存的对象为数组，内容为host,ipv6,ipv4来访IP,at当前时间戳
5. 初始化密码，检查密码时如果缓存里没有保存的密码，就将当前要校验的密码作为密码保存，同时进入密码验证正确的流程,密码保存在配置文件中
6. 密码保存和校验密码使用相同的算法bcrypt
7. 缓存类以go读写内存来实现，内容24小时后过期
8. 只接受utf-8编码，参数host长度不超过16,ipv6不超过39
9. 缓存以host为键名，存入前检查ip4和ip6内容和缓存的是否不同，有变化才存入，如果变化的是ipv6，调用阿里云修改域名解析
10. 修改解析前从配置文件读取相应host的配置，没有则不更新
11. 配置文件使用toml,保存有AccessKey ID和AccessKey Secret，对应域名，支持多个域名和多个ali帐号

## 编译为linux64位名为go_ddns
```
GOOS=linux GOARCH=amd64 go build -o go_ddns main.go
```

## Dockerfile
### 生成并导出镜象
```
sudo docker build . --tag go-ddns
sudo docker save go-ddns >./go-ddns.img
sudo zip ./go-ddns.img.zip ./go-ddns.img
sudo rm ./go-ddns.img
```
### 导入镜像
```
unzip go-ddns.img.zip
sudo docker load <go-ddns.img
rm go-ddns.img*
```
### 运行镜象
```
docker run --restart=always --name go-ddns -p 8080:8080 --log-opt max-size=1m -d go-ddns
# 映射用法示例
docker run --restart=always --name go-ddns -p 8080:8080 -v ./config.toml:/app/config.toml --log-opt max-size=1m -d go-ddns
```
## 以下为windows上报代码示例

```
@echo off
rem windows下获取ipv6地址上报，保存为cmd文件并在计划任务里登记，注意window文件编码,错误时会识别不出中文的关键字"临时"
rem 以下两个现要按需修改：YOUR-HOST-NAME和https://example.com/
SETLOCAL ENABLEDELAYEDEXPANSION
set host=YOUR-HOST-NAME
set firstIPv6=0
rem 中文环境里面是"临时"
for /f "tokens=5 delims= " %%A in ('netsh interface ipv6 show addresses ^| findstr "临时"') do (
set firstIPv6=%%A
goto sendRequest
)


:sendRequest
if "!firstIPv6!"=="0" (
goto end
)

curl -X POST "https://example.com/" -H "Content-Type: application/json" -d "{ \"host\": \"!host!\", \"ipv6\": \"!firstIPv6!\" }"
:end
ENDLOCAL
```
 1. Windows11下 计算机管理->系统工具->任务计划程序->创建任务(不是基本任务)
 2. 常规选项卡：填写名称(随意)/勾选不管用户是否登录都要运行/更改用户为system
 3. 触发器：新建触发器->开始任务：启动时/延迟任务时间30秒/重复间隔1小时，持续时间无限期/已启用
 4. 操作：新建->启动程序/程序或脚本(浏览到脚本文件)
 5. 设置：去除勾选"如果任务超过以下时间，停止任务"