## Introduction

Related data and tools of New Version of DDG P2P minning botnet

**Refer**:

1. [DDG botnet, round X, is there an ending?](https://blog.netlab.360.com/an-update-on-the-ddg-botnet/)
2. [DDG的新征程——自研P2P协议构建混合P2P网络](https://blog.netlab.360.com/ddg-upgrade-to-new-p2p-hybrid-model/)

### tracker

**tracker.elf** is a pre-built ELF Binary file, you can just run it as:


```
$ ./tracker.elf -s seeds.list
```

**seeds.list** is a list file contains seed nodes address( `<ip:port>` ) of P2P network, it can extracted from DDG binary sample. Related tools：[**dec_seeds.go**](./tools/dec_seeds.go)

It outputs tracking results to STDOUT, and will take about 130 minutes to finish its tracking job. After finishing tracking, **Tracker.elf** will save all the tracked nodes address to a file named as **nodes_tracked.list**.

File information of **tracker.elf**:

```
$ file tracker.elf
tracker.elf: ELF 64-bit LSB executable, x86-64, version 1 (SYSV), statically linked, stripped

$ sha256sum tracker.elf
b104560ad7511945a353be1fe9134beca82ae12a7c4a9f0ae4dbce1c5371f3bb  tracker.elf
```

Usage of **tracker.elf**:

```
$ ./tracker.elf -h
Usage: ./tracker -s <Seed List File>

Options:
    -h help
        Print this help
    -s seeds-file
        Path of nodes <seeds-file.>
```

