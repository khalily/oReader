FROM python:3.11-slim

WORKDIR /app

# Install system dependencies for MinerU/PDF processing
# - libgl1, libglib2.0: OpenCV dependencies (used by MinerU)
# - libgomp1: OpenMP (used by NumPy/PyTorch)
RUN apt-get update && apt-get install -y --no-install-recommends \
    libgl1-mesa-glx \
    libglib2.0-0 \
    libgomp1 \
    && rm -rf /var/lib/apt/lists/*

# Copy and install Python dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Create minimal magic-pdf config (required by magic_pdf library)
# CPU-only mode, formula/table recognition disabled for Docker environments
RUN echo '{"layout-config":{"model":"doclayout_yolo"},"formula-config":{"mfd_model":"yolo_v8_mfd","mfr_model":"unimernet_small","enable":false},"table-config":{"model":"tablemaster","enable":false,"max_time":60},"device-mode":"cpu"}' > /root/magic-pdf.json

# Copy converter source
COPY . .

EXPOSE 50051

# Health check via TCP socket
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD python -c "import socket; s=socket.socket(); s.settimeout(5); s.connect(('localhost', 50051)); s.close()" || exit 1

CMD ["python", "server.py"]
