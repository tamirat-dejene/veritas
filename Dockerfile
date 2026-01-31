FROM python:3.12-slim

ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1

WORKDIR /app

# 1. System dependencies - added g++ as some C++ extensions might require it
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    gcc \
    g++ \
    python3-dev \
    && rm -rf /var/lib/apt/lists/*

# 2. Pre-install heavy dependencies and build tools
RUN pip install --no-cache-dir --upgrade pip && \
    pip install --no-cache-dir \
    pytest pytest-doctestplus pytest-xdist pytest-arraydiff \
    pandas numpy astropy \
    setuptools_scm cython extension-helpers

# 3. Clone and checkout in one layer to keep the image slim
RUN git clone https://github.com/sunpy/sunpy.git . && \
    git checkout a1a081a

# 4. Install SunPy from source 
# We use --no-build-isolation because we pre-installed the build tools in Step 2. This makes the build faster by avoiding redundant installations.
RUN pip install --no-cache-dir --no-build-isolation -e .

# Ensure the local directory is prioritized
ENV PYTHONPATH=/app

CMD ["pytest", "sunpy/time/tests/test_time.py"]