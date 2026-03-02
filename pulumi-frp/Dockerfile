FROM debian:13

RUN apt-get update && apt-get install -y \
    wget \
    python3 \
    && rm -rf /var/lib/apt/lists/*

RUN wget https://github.com/fatedier/frp/releases/download/v0.67.0/frp_0.67.0_linux_amd64.tar.gz \
    && tar -xzf frp_0.67.0_linux_amd64.tar.gz \
    && mv frp_0.67.0_linux_amd64/frpc /usr/local/bin/ \
    && rm -rf frp_0.67.0_linux_amd64*

RUN mkdir /www && echo "<html><body><h1>Hello from frp client!</h1></body></html>" > /www/index.html

COPY frpc.toml /etc/frpc.toml
COPY frpc.crt /etc/frpc.crt
COPY frpc.key /etc/frpc.key
COPY frp-ca.crt /etc/frp-ca.crt

CMD ["/bin/sh", "-c", "python3 -m http.server 8080 -d /www & frpc -c /etc/frpc.toml"]
