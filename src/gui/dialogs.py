from PyQt6.QtWidgets import QDialog, QVBoxLayout, QTextEdit, QPushButton
from PyQt6.QtGui import QFont

class LogDialog(QDialog):
    def __init__(self):
        super().__init__()
        self.setWindowTitle("Proxy Server Logs")
        self.setGeometry(100, 100, 800, 600)
        
        layout = QVBoxLayout()
        
        self.log_text = QTextEdit()
        self.log_text.setReadOnly(True)
        self.log_text.setFont(QFont("Monospace", 10))
        layout.addWidget(self.log_text)
        
        self.clear_button = QPushButton("Clear Logs")
        self.clear_button.clicked.connect(self.clear_logs)
        layout.addWidget(self.clear_button)
        
        self.setLayout(layout)
    
    def add_log(self, message):
        self.log_text.append(message)
        self.log_text.verticalScrollBar().setValue(self.log_text.verticalScrollBar().maximum())
    
    def clear_logs(self):
        self.log_text.clear()

class GitDialog(QDialog):
    def __init__(self, git_executor):
        super().__init__()
        self.git_executor = git_executor
        self.setWindowTitle("Git Command Executor")
        self.setGeometry(100, 100, 600, 400)
        
        layout = QVBoxLayout()
        
        self.command_input = QTextEdit()
        self.command_input.setPlaceholderText("Enter git command (e.g., clone https://github.com/user/repo.git)")
        layout.addWidget(self.command_input)
        
        self.execute_button = QPushButton("Execute")
        self.execute_button.clicked.connect(self.execute_command)
        layout.addWidget(self.execute_button)
        
        self.output_text = QTextEdit()
        self.output_text.setReadOnly(True)
        layout.addWidget(self.output_text)
        
        self.setLayout(layout)
    
    def execute_command(self):
        command_text = self.command_input.toPlainText().strip()
        if not command_text:
            return
        
        # 解析命令
        parts = command_text.split()
        if parts[0] != 'git':
            parts.insert(0, 'git')
        
        # 执行命令
        def log_callback(message):
            self.output_text.append(message)
            self.output_text.verticalScrollBar().setValue(self.output_text.verticalScrollBar().maximum())
        
        self.git_executor.log_callback = log_callback
        self.git_executor.execute(parts)

class VerifyDialog(QDialog):
    def __init__(self):
        super().__init__()
        self.setWindowTitle("Proxy Verification")
        self.setGeometry(100, 100, 600, 400)
        
        layout = QVBoxLayout()
        
        self.result_text = QTextEdit()
        self.result_text.setReadOnly(True)
        layout.addWidget(self.result_text)
        
        self.close_button = QPushButton("Close")
        self.close_button.clicked.connect(self.close)
        layout.addWidget(self.close_button)
        
        self.setLayout(layout)
    
    def add_result(self, message):
        self.result_text.append(message)
        self.result_text.verticalScrollBar().setValue(self.result_text.verticalScrollBar().maximum())
