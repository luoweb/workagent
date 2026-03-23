from PyQt6.QtWidgets import QSystemTrayIcon, QMenu
from PyQt6.QtGui import QAction, QIcon

class SystemTray(QSystemTrayIcon):
    def __init__(self, app):
        super().__init__()
        self.app = app
        self.setIcon(QIcon.fromTheme('network-server'))
        self.create_menu()
        self.show()
    
    def create_menu(self):
        self.menu = QMenu()
        
        # 监听地址
        self.menu.addSection("Listening Addresses")
        
        # HTTP地址
        http_port = self.app.config.get('ports', {}).get('http', 8080)
        http_address = f"http://localhost:{http_port}"
        http_action = QAction(f"HTTP: {http_address}", self)
        http_action.triggered.connect(lambda: self.app.copy_to_clipboard(http_address))
        self.menu.addAction(http_action)
        
        # HTTPS地址
        https_port = self.app.config.get('ports', {}).get('https', 8443)
        https_address = f"https://localhost:{https_port}"
        https_action = QAction(f"HTTPS: {https_address}", self)
        https_action.triggered.connect(lambda: self.app.copy_to_clipboard(https_address))
        self.menu.addAction(https_action)
        
        # SSH地址
        ssh_port = self.app.config.get('ports', {}).get('ssh', 2222)
        ssh_address = f"ssh://localhost:{ssh_port}"
        ssh_action = QAction(f"SSH: {ssh_address}", self)
        ssh_action.triggered.connect(lambda: self.app.copy_to_clipboard(ssh_address))
        self.menu.addAction(ssh_action)
        
        # SOCKS5地址
        socks5_port = self.app.config.get('ports', {}).get('socks5', 1080)
        socks5_address = f"socks5://localhost:{socks5_port}"
        socks5_action = QAction(f"SOCKS5: {socks5_address}", self)
        socks5_action.triggered.connect(lambda: self.app.copy_to_clipboard(socks5_address))
        self.menu.addAction(socks5_action)
        
        # TCP地址
        tcp_port = self.app.config.get('ports', {}).get('tcp', 9090)
        tcp_address = f"tcp://localhost:{tcp_port}"
        tcp_action = QAction(f"TCP: {tcp_address}", self)
        tcp_action.triggered.connect(lambda: self.app.copy_to_clipboard(tcp_address))
        self.menu.addAction(tcp_action)
        
        self.menu.addSeparator()
        
        # 验证代理
        verify_action = QAction("Verify Proxy", self)
        verify_action.triggered.connect(self.app.verify_proxy)
        self.menu.addAction(verify_action)
        
        # 查看日志
        log_action = QAction("View Logs", self)
        log_action.triggered.connect(self.app.show_logs)
        self.menu.addAction(log_action)
        
        # Git命令
        git_action = QAction("Git Command", self)
        git_action.triggered.connect(self.app.show_git)
        self.menu.addAction(git_action)
        
        # 退出
        quit_action = QAction("Quit", self)
        quit_action.triggered.connect(self.app.quit_app)
        self.menu.addAction(quit_action)
        
        self.setContextMenu(self.menu)
