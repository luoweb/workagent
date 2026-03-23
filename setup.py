from setuptools import setup, find_packages

setup(
    name="git-proxy-server",
    version="0.1.0",
    packages=find_packages(),
    include_package_data=True,
    install_requires=[
        "PyQt6",
    ],
    entry_points={
        "console_scripts": [
            "git-proxy = main:main",
        ],
    },
    classifiers=[
        "Programming Language :: Python :: 3",
        "License :: OSI Approved :: MIT License",
        "Operating System :: OS Independent",
    ],
    python_requires='>=3.7',
)
