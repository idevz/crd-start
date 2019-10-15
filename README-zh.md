export GO111MODULE=on
cobra init crd-start --pkg-name github.com/idevz/crd-start
cd crd-start
go mod init github.com/idevz/crd-start
go mod tidy

基于 cobra 搭建项目骨架
编写 CRD，CR
根据 CRD 和 CR 在 `pkg/apis` 目录下编写相应的 API 对象描述文件，并添加相应的代码生成注释
使用 Kubernetes 提供的代码生成工具，生成相应的 deepcopy、client、lister、informer 包 `pkg/client`