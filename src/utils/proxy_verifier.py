import socket
import threading

class ProxyVerifier:
    def __init__(self, config, log_callback):
        self.config = config
        self.log_callback = log_callback
    
    def verify_all(self):
        results = []
        threads = []
        
        # 验证HTTP代理
        http_port = self.config.get('ports', {}).get('http', 8080)
        t = threading.Thread(target=self.verify_http, args=(http_port, results))
        threads.append(t)
        t.start()
        
        # 验证HTTPS代理
        https_port = self.config.get('ports', {}).get('https', 8443)
        t = threading.Thread(target=self.verify_https, args=(https_port, results))
        threads.append(t)
        t.start()
        
        # 验证SOCKS5代理
        socks5_port = self.config.get('ports', {}).get('socks5', 1080)
        t = threading.Thread(target=self.verify_socks5, args=(socks5_port, results))
        threads.append(t)
        t.start()
        
        # 验证TCP代理
        tcp_port = self.config.get('ports', {}).get('tcp', 9090)
        t = threading.Thread(target=self.verify_tcp, args=(tcp_port, results))
        threads.append(t)
        t.start()
        
        # 等待所有线程完成
        for t in threads:
            t.join()
        
        return results
    
    def verify_http(self, port, results):
        try:
            sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            sock.settimeout(2)
            result = sock.connect_ex(('localhost', port))
            if result == 0:
                results.append(('HTTP', port, '✓'))
                self.log_callback(f"HTTP proxy on port {port} is working")
            else:
                results.append(('HTTP', port, '✗'))
                self.log_callback(f"HTTP proxy on port {port} is not working")
            sock.close()
        except Exception as e:
            results.append(('HTTP', port, '✗'))
            self.log_callback(f"Error verifying HTTP proxy: {e}")
    
    def verify_https(self, port, results):
        try:
            sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            sock.settimeout(2)
            result = sock.connect_ex(('localhost', port))
            if result == 0:
                results.append(('HTTPS', port, '✓'))
                self.log_callback(f"HTTPS proxy on port {port} is working")
            else:
                results.append(('HTTPS', port, '✗'))
                self.log_callback(f"HTTPS proxy on port {port} is not working")
            sock.close()
        except Exception as e:
            results.append(('HTTPS', port, '✗'))
            self.log_callback(f"Error verifying HTTPS proxy: {e}")
    
    def verify_socks5(self, port, results):
        try:
            sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            sock.settimeout(2)
            result = sock.connect_ex(('localhost', port))
            if result == 0:
                # 发送SOCKS5握手请求
                sock.sendall(b'\x05\x01\x00')
                response = sock.recv(2)
                if len(response) == 2 and response[0] == 5 and response[1] == 0:
                    results.append(('SOCKS5', port, '✓'))
                    self.log_callback(f"SOCKS5 proxy on port {port} is working")
                else:
                    results.append(('SOCKS5', port, '✗'))
                    self.log_callback(f"SOCKS5 proxy on port {port} is not working")
            else:
                results.append(('SOCKS5', port, '✗'))
                self.log_callback(f"SOCKS5 proxy on port {port} is not working")
            sock.close()
        except Exception as e:
            results.append(('SOCKS5', port, '✗'))
            self.log_callback(f"Error verifying SOCKS5 proxy: {e}")
    
    def verify_tcp(self, port, results):
        try:
            sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            sock.settimeout(2)
            result = sock.connect_ex(('localhost', port))
            if result == 0:
                results.append(('TCP', port, '✓'))
                self.log_callback(f"TCP proxy on port {port} is working")
            else:
                results.append(('TCP', port, '✗'))
                self.log_callback(f"TCP proxy on port {port} is not working")
            sock.close()
        except Exception as e:
            results.append(('TCP', port, '✗'))
            self.log_callback(f"Error verifying TCP proxy: {e}")
