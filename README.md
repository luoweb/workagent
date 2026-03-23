# Git Proxy Server

基于PyQt6框架实现的服务端代理，支持git代码仓库请求的https、http、ssh三种协议的代理程序。

## 功能特点

- **多协议支持**：支持HTTP、HTTPS、SSH三种协议的代理
- **系统托盘**：运行时在系统托盘显示图标，方便操作
- **Git命令执行**：内置Git命令执行功能，可以直接在应用中执行Git操作
- **静默后台**：应用在后台运行，不干扰用户正常工作
- **多平台支持**：可在Windows、macOS、Linux等平台运行

## 安装方法

### 方法一：使用pip安装

```bash
pip3 install -e .
```

### 方法二：直接运行

```bash
python3 main.py
```

## 配置说明

应用启动时会自动创建`config.json`配置文件，默认配置如下：

```json
{
  "ports": {
    "http": 8080,
    "https": 8443,
    "ssh": 2222
  }
}
```

可以根据需要修改端口号。

## 使用方法

1. 启动应用后，系统托盘会显示一个网络服务器图标
2. 右键点击图标，选择"View Logs"查看代理服务器日志
3. 选择"Git Command"打开Git命令执行窗口，输入Git命令并执行
4. 选择"Quit"退出应用

## 代理服务器功能

应用会在指定端口启动HTTP、HTTPS、SSH代理服务器，用于处理Git仓库请求。当前实现为基础框架，具体的代理逻辑需要根据实际需求进行扩展。

## 打包说明

### 使用PyInstaller打包

1. 安装PyInstaller

```bash
pip3 install pyinstaller
```

2. 打包应用

```bash
# Windows
pyinstaller --onefile --windowed --name git-proxy main.py

# macOS
pyinstaller --onefile --windowed --name git-proxy main.py

# Linux
pyinstaller --onefile --name git-proxy main.py
```

## 注意事项

- 确保使用的端口没有被其他程序占用
- 运行SSH代理可能需要管理员权限
- 本应用为基础框架，实际使用时需要根据具体需求扩展代理逻辑
