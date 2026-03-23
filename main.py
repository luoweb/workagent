import sys
import os
import json

from PyQt6.QtWidgets import QApplication
from PyQt6.QtCore import Qt

from src.proxy.server import ProxyServer
from src.gui.tray import SystemTray
from src.gui.dialogs import LogDialog, GitDialog, VerifyDialog
from src.utils.git_executor import GitCommandExecutor
from src.utils.proxy_verifier import ProxyVerifier

class App(QApplication):
    def __init__(self, argv):
        super().__init__(argv)
        
        # 加载配置
        self.config = self.load_config()
        
        # 初始化代理服务器
        self.proxy_server = ProxyServer(self.config)
        self.proxy_server.log_signal.connect(self.on_log)
        
        # 初始化Git命令执行器
        self.git_executor = GitCommandExecutor(self.on_log)
        
        # 初始化日志对话框
        self.log_dialog = LogDialog()
        
        # 初始化Git对话框
        self.git_dialog = GitDialog(self.git_executor)
        
        # 初始化验证对话框
        self.verify_dialog = VerifyDialog()
        
        # 创建系统托盘
        self.tray = SystemTray(self)
        
        # 启动代理服务器
        self.proxy_server.start()
    
    def load_config(self):
        config_path = os.path.join(os.path.dirname(__file__), 'config.json')
        default_config = {
            'ports': {
                'http': 8080,
                'https': 8443,
                'ssh': 2222,
                'socks5': 1080,
                'tcp': 9090
            }
        }
        
        if os.path.exists(config_path):
            try:
                with open(config_path, 'r') as f:
                    return json.load(f)
            except Exception as e:
                print(f"Error loading config: {e}")
                return default_config
        else:
            # 创建默认配置文件
            with open(config_path, 'w') as f:
                json.dump(default_config, f, indent=2)
            return default_config
    
    def show_logs(self):
        self.log_dialog.show()
    
    def show_git(self):
        self.git_dialog.show()
    
    def verify_proxy(self):
        self.verify_dialog.result_text.clear()
        verifier = ProxyVerifier(self.config, self.on_log)
        results = verifier.verify_all()
        
        for protocol, port, status in results:
            self.verify_dialog.add_result(f"{protocol}: localhost:{port} - {status}")
        
        self.verify_dialog.show()
    
    def copy_to_clipboard(self, text):
        clipboard = self.clipboard()
        clipboard.setText(text)
        self.tray.showMessage("Git Proxy", f"Copied to clipboard: {text}", self.tray.Information, 2000)
    
    def on_log(self, message):
        self.log_dialog.add_log(message)
    
    def quit_app(self):
        self.proxy_server.stop()
        self.quit()

def main():
    # 确保应用在后台运行
    if sys.platform == 'darwin':
        # macOS 特殊处理
        os.environ['QT_MAC_WANTS_LAYER'] = '1'
    
    app = App(sys.argv)
    # 隐藏主窗口
    app.setQuitOnLastWindowClosed(False)
    sys.exit(app.exec())

if __name__ == '__main__':
    main()
