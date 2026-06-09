FROM python:3.12-slim

WORKDIR /app

# Install SDK from repo root (Dockerfile context is repo root)
COPY drivers/sdk /sdk
COPY drivers/ethernet-ip/requirements.txt .
RUN pip install --no-cache-dir /sdk && \
    pip install --no-cache-dir pycomm3>=1.2.13 && \
    pip install --no-cache-dir fastapi uvicorn[standard] pydantic

COPY drivers/ethernet-ip/driver.py .

EXPOSE 8080

# EtherNet/IP communicates on TCP 44818 from the driver to the PLC.
# If the PLC is on a network unreachable from pod CIDR, set hostNetwork: true
# in the deployment (see infra/drivers/ethernet-ip/deployment.yaml).

CMD ["python", "driver.py"]
