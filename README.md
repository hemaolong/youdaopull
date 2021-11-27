#### 解决什么问题

批量导出有道云笔记到本地备份

#### 使用

- windows 下直接运行
  下载 bin 中文件，执行 run.bat(windows 乱码)，如果装有 linux shell（例如 gitbash），可以直接运行
- linux 等其他平台请下载代码自行编译

#### features

- [x] 微信登陆
- [x] cookie 登陆，一定时间内多次登陆不需要每次扫描二维码
- [x] 遍历拉取所有文件到本地
- [ ] 指定目录拉取（只能是第一层目录，适用于日志特别多的朋友)
- [x] 本地缓存，只拉取变化的文件  
       可以正确处理文件移动，如果有道云笔记文件移动，本地会随之变化而不会出现重复
  - 本地会缓存所有文件信息，如果本地文件跟线上一致不会重复拉取
- [ ] 多线程同时拉？  
       本人笔记量小，图片更少，暂时没有支持多线程拉取

##### 已知问题

- windows terminal 不能正确显示日志颜色

#### 其他

##### 拉取过程

1. 使用微信二维码登陆，因为本人有道云笔记没有使用用户名、密码
2. 重组检查本地缓存的 meta 文件
3. 根据本地缓信息增量拉取文件保存到本地目录

##### 网上已有导出工具，为什么重复造轮子

- 网上都是 python 版本，使用需要安装运行环境，对不懂编程的人不是非常方便
- 平时很少接触 web 编程，顺便练练手

- 参考工具
  - [ynote2hexo](https://github.com/liuyi12138/ynote2hexo)
