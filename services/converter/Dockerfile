ARG PYTHON_VERSION
FROM python:${PYTHON_VERSION}-slim

COPY --from=ghcr.io/astral-sh/uv:latest /uv /usr/local/bin/uv

WORKDIR /app

RUN sed -i 's/deb.debian.org/mirrors.aliyun.com/g' /etc/apt/sources.list.d/debian.sources
RUN apt-get update && apt-get install -y --no-install-recommends \
    libgl1 libglib2.0-0 libgomp1 && rm -rf /var/lib/apt/lists/*

COPY services/converter/pyproject.toml .
RUN uv pip install --system --no-cache --prerelease=allow -i https://mirrors.aliyun.com/pypi/simple/ --trusted-host mirrors.aliyun.com .

RUN echo '{"layout-config":{"model":"doclayout_yolo"},"formula-config":{"mfd_model":"yolo_v8_mfd","mfr_model":"unimernet_small","enable":false},"table-config":{"model":"tablemaster","enable":false,"max_time":60},"device-mode":"cpu"}' > /root/magic-pdf.json

# Copy proto generated Python code (context is project root)
# server.py 在 /app/src/，其 ../../../proto/python 解析为 /proto/python
COPY proto/python/ /proto/python/

COPY services/converter/src/ ./src/

EXPOSE 50051
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD python -c "import socket; s=socket.socket(); s.settimeout(5); s.connect(('localhost', 50051)); s.close()" || exit 1

CMD ["python", "src/server.py"]
