import pytest
from src.utils.proxy_verifier import ProxyVerifier

class TestProxyVerifier:
    def test_verify_all(self):
        config = {
            'ports': {
                'http': 8080,
                'https': 8443,
                'socks5': 1080,
                'tcp': 9090
            }
        }
        
        def log_callback(message):
            pass
        
        verifier = ProxyVerifier(config, log_callback)
        results = verifier.verify_all()
        
        # 验证返回结果格式正确
        assert isinstance(results, list)
        for result in results:
            assert len(result) == 3
            assert isinstance(result[0], str)
            assert isinstance(result[1], int)
            assert isinstance(result[2], str)

if __name__ == '__main__':
    pytest.main(['-v', __file__])
