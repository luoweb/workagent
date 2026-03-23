import subprocess

class GitCommandExecutor:
    def __init__(self, log_callback):
        self.log_callback = log_callback
    
    def execute(self, command, cwd=None):
        try:
            self.log_callback(f"Executing git command: {' '.join(command)}")
            process = subprocess.Popen(
                command, 
                cwd=cwd, 
                stdout=subprocess.PIPE, 
                stderr=subprocess.PIPE, 
                text=True
            )
            stdout, stderr = process.communicate()
            if stdout:
                self.log_callback(f"Git output: {stdout.strip()}")
            if stderr:
                self.log_callback(f"Git error: {stderr.strip()}")
            return process.returncode, stdout, stderr
        except Exception as e:
            self.log_callback(f"Error executing git command: {e}")
            return -1, "", str(e)
