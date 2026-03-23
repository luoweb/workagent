import socket
import threading
from datetime import datetime
from PyQt6.QtCore import QObject, pyqtSignal

class ProxyServer(QObject):
    log_signal = pyqtSignal(str)
    
    def __init__(self, config):
        super().__init__()
        self.config = config
        self.servers = {}
        self.running = False
    
    def start(self):
        self.running = True
        for protocol, port in self.config.get('ports', {}).items():
            if protocol in ['http', 'https', 'ssh', 'socks5', 'tcp']:
                server_thread = threading.Thread(target=self.run_server, args=(protocol, port))
                server_thread.daemon = True
                server_thread.start()
                self.servers[protocol] = server_thread
                self.log(f"Started {protocol} proxy server on port {port}")
    
    def stop(self):
        self.running = False
        for protocol, server in self.servers.items():
            if server.is_alive():
                server.join(timeout=1)
        self.log("Stopped all proxy servers")
    
    def run_server(self, protocol, port):
        try:
            server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            server_socket.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
            server_socket.bind(('0.0.0.0', port))
            server_socket.listen(5)
            
            while self.running:
                try:
                    client_socket, client_address = server_socket.accept()
                    client_thread = threading.Thread(target=self.handle_client, args=(protocol, client_socket, client_address))
                    client_thread.daemon = True
                    client_thread.start()
                except socket.timeout:
                    continue
                except Exception as e:
                    self.log(f"Error accepting connection: {e}")
                    continue
        except Exception as e:
            self.log(f"Error starting {protocol} server: {e}")
    
    def handle_client(self, protocol, client_socket, client_address):
        try:
            self.log(f"{protocol.upper()} Connection from {client_address}")
            # 读取请求并处理
            data = client_socket.recv(4096)
            if data:
                self.log(f"Received {len(data)} bytes from {client_address}")
                # 根据协议类型进行不同的处理
                if protocol == 'http':
                    response = b'HTTP/1.1 200 OK\r\nContent-Length: 12\r\n\r\nHello World!'
                elif protocol == 'socks5':
                    # SOCKS5 协议处理
                    response = self.handle_socks5(data)
                else:
                    response = b'Proxy response'
                client_socket.sendall(response)
        except Exception as e:
            self.log(f"Error handling client: {e}")
        finally:
            client_socket.close()
    
    def handle_socks5(self, data):
        # 简单的 SOCKS5 握手处理
        if data[0] == 5:  # SOCKS5 版本
            # 支持的认证方法
            return b'\x05\x00'  # 无需认证
        return b'\x05\xff'  # 无可用认证方法
    
    def log(self, message):
        timestamp = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
        log_message = f"[{timestamp}] {message}"
        self.log_signal.emit(log_message)
        print(log_message)
