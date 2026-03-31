#!/bin/bash
set -e

export MODELS_DIR="${MODELS_DIR:-/root/.cache/magic-pdf/models}"
MARKER="$MODELS_DIR/.downloaded"

if [ ! -f "$MARKER" ]; then
    echo "Downloading MinerU models to $MODELS_DIR ..."
    python -c "
import os, sys, shutil

models_dir = os.environ['MODELS_DIR']
os.makedirs(models_dir, exist_ok=True)

# Download all model subsets we need (layout, mfd, mfr, ocr)
# See: magic_pdf/resources/model_config/model_configs.yaml
allow_patterns = [
    'models/Layout/YOLO/*',
    'models/MFD/YOLO/*',
    'models/MFR/unimernet_hf_small_2503/*',
    'models/OCR/paddleocr_torch/*',
]

# Try ModelScope first (best for China network)
try:
    from modelscope import snapshot_download
    print('Using modelscope to download models...')
    repo_dir = snapshot_download(
        'OpenDataLab/PDF-Extract-Kit-1.0',
        allow_patterns=allow_patterns,
    )
except (ImportError, Exception) as e:
    if 'ImportError' not in type(e).__name__:
        print(f'ModelScope download failed: {e}, falling back to HuggingFace...')
    # Fallback to HuggingFace mirror
    os.environ.setdefault('HF_ENDPOINT', 'https://hf-mirror.com')
    from huggingface_hub import snapshot_download as hf_download
    print('Using huggingface mirror to download models...')
    repo_dir = hf_download(
        'opendatalab/PDF-Extract-Kit-1.0',
        allow_patterns=allow_patterns,
    )

# The downloaded repo has a 'models/' subdirectory containing Layout/, MFD/, MFR/
src_models = os.path.join(repo_dir, 'models')
if os.path.isdir(src_models):
    # Copy model files to the configured models-dir
    for subdir in os.listdir(src_models):
        src = os.path.join(src_models, subdir)
        dst = os.path.join(models_dir, subdir)
        if os.path.isdir(src):
            if os.path.exists(dst):
                shutil.rmtree(dst)
            shutil.copytree(src, dst)
            print(f'  Copied {subdir}/')
else:
    # If no 'models/' prefix, copy directly
    for item in os.listdir(repo_dir):
        src = os.path.join(repo_dir, item)
        dst = os.path.join(models_dir, item)
        if os.path.isdir(src) and not os.path.exists(dst):
            shutil.copytree(src, dst)
            print(f'  Copied {item}/')

# The PDF-Extract-Kit repo only has OCR v4/v5 models, but magic_pdf 1.3.12
# requires v3 det model (ch_PP-OCRv3_det_infer.pth) for all Chinese profiles.
# Download it from an earlier repo commit before it was deleted.
v3_det_path = os.path.join(models_dir, 'OCR/paddleocr_torch/ch_PP-OCRv3_det_infer.pth')
if not os.path.exists(v3_det_path):
    print('Downloading ch_PP-OCRv3_det_infer.pth (required by magic_pdf but removed from latest repo)...')
    os.makedirs(os.path.dirname(v3_det_path), exist_ok=True)
    os.environ.setdefault('HF_ENDPOINT', 'https://hf-mirror.com')
    from huggingface_hub import hf_hub_download
    downloaded = hf_hub_download(
        'opendatalab/PDF-Extract-Kit-1.0',
        'models/OCR/paddleocr_torch/ch_PP-OCRv3_det_infer.pth',
        revision='782e787d46ed9b52253af6c1f69cdfcc76583e8d',
    )
    shutil.copy2(downloaded, v3_det_path)
    print(f'  OK: ch_PP-OCRv3_det_infer.pth ({os.path.getsize(v3_det_path) / (1024*1024):.1f} MB)')

# Download layoutreader model (reading order排序模型).
# Default fallback path: /root/.cache/modelscope/hub/ppaanngggg/layoutreader
layoutreader_dir = os.path.expanduser('~/.cache/modelscope/hub/ppaanngggg/layoutreader')
if not os.path.exists(os.path.join(layoutreader_dir, 'config.json')):
    print('Downloading layoutreader model (reading order)...')
    os.makedirs(layoutreader_dir, exist_ok=True)
    os.environ.setdefault('HF_ENDPOINT', 'https://hf-mirror.com')
    try:
        from modelscope import snapshot_download as ms_download
        ms_download('ppaanngggg/layoutreader')
        # ModelScope caches at ~/.cache/modelscope/hub/models/ppaanngggg/layoutreader
        # but magic_pdf looks at ~/.cache/modelscope/hub/ppaanngggg/layoutreader
        actual_dir = os.path.expanduser('~/.cache/modelscope/hub/models/ppaanngggg/layoutreader')
        if os.path.exists(actual_dir) and not os.path.exists(layoutreader_dir):
            os.makedirs(os.path.dirname(layoutreader_dir), exist_ok=True)
            os.symlink(actual_dir, layoutreader_dir)
            print(f'  Linked {actual_dir} -> {layoutreader_dir}')
        print(f'  OK: layoutreader model downloaded')
    except Exception as e:
        print(f'  WARNING: layoutreader download failed ({e}), will try at runtime via HF_ENDPOINT')

print('Model download completed.')
# Verify key model files exist
expected = [
    'Layout/YOLO/doclayout_yolo_docstructbench_imgsz1280_2501.pt',
    'MFD/YOLO/yolo_v8_ft.pt',
    'OCR/paddleocr_torch/ch_PP-OCRv3_det_infer.pth',
]
for f in expected:
    path = os.path.join(models_dir, f)
    if os.path.exists(path):
        size_mb = os.path.getsize(path) / (1024 * 1024)
        print(f'  OK: {f} ({size_mb:.1f} MB)')
    else:
        print(f'  WARNING: {f} not found!')
"
    touch "$MARKER"
    echo "Models downloaded successfully."
else
    echo "Models already exist at $MODELS_DIR, skipping download."
fi

exec "$@"
