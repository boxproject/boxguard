boxgd作为签名机(voucher)的伴生程序用来为签名机保驾护航，其使用也非常地简单。

因为此程序是在保护voucher正常运行的，所以启动前请先查看voucher是否是正常运行的，如果是请记下
voucher的进程号即pid字符串，将其pid写入config.toml中的protectId中，如：
ProtectId="34353"

config.toml字段释义：
WaitSeconds ： 程序正式启动前的等待延时
EnablePfctl ：是否启用mac防火墙
EnableProcGuard：是否启用进程监控防护
AllowUser ：调试时的备用项
ProtectId：保护的进程号，即voucher的pid
WhiteList:进程白名单数组
[Monitor]
Users : 允许登录本mac的用户数
PrcName : voucher服务名称


编译：
make build
编译后会生成build


使用方法：
cd /build/bin
1.安装服务

```bash
sudo ./boxgd install
```

2.启动服务：

```bash
sudo ./boxgd start
```



3.停止服务:

```bash
sudo ./boxgd stop
```



4.查看服务状态：

```bash
sudo ./boxgd status
```



5.卸载服务：

```Bash
sudo ./boxgd remove
```
the end
