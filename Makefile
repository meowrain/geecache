# 定义目标程序名称
TARGET = mikucache

# 定义源文件目录
SRC_DIR = .

# 定义源文件
SRCS = $(wildcard $(SRC_DIR)/*.go)

# 定义编译目标
all: linux windows mac

# 编译Linux版本
linux:
	GOOS=linux GOARCH=amd64 go build -o $(TARGET)-linux $(SRCS)

# 编译Windows版本
windows:
	GOOS=windows GOARCH=amd64 go build -o $(TARGET)-windows.exe $(SRCS)

# 编译Mac版本
mac:
	GOOS=darwin GOARCH=amd64 go build -o $(TARGET)-mac $(SRCS)

# 清理生成的文件
clean:
	rm -f $(TARGET)-linux $(TARGET)-windows.exe $(TARGET)-mac
