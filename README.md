# AI Operator

The AI Operator is a Kubernetes operator that simplifies running AI fine-tuning jobs on Kubernetes clusters. It manages resources needed to fine-tune large language models, including model downloads, persistence volumes, and Hugging Face authentication.

## Overview

This operator introduces a new Custom Resource Definition (CRD) called `Job` that wraps and automates:

- Setting up persistent storage for model files
- Managing Hugging Face authentication tokens securely 
- Downloading LLM models from Hugging Face
- Running fine-tuning jobs with GPU support
- Managing the lifecycle of resources

## Installation

### Prerequisites
- Kubernetes cluster with GPU support
- kubectl configured to access your cluster
- NVIDIA runtime configured on nodes

Install the operator:

```bash
kubectl apply -f https://raw.githubusercontent.com/re-cinq/ai-operator/main/dist/install.yaml
```

## Usage

### 1. Create a Hugging Face Token Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: hf-token
  namespace: default
data:
  token: <base64-encoded-token>
```

### 2. Create an AI Job

```yaml
apiVersion: ai.re-cinq.com/v1
kind: Job
metadata:
  name: finetune-job
spec:
  # NVIDIA runtime class for GPU access
  runtimeClassName: "nvidia"

  # Container image with training code
  image: "silentehrec/torchtune:latest" 

  # Model to download from Hugging Face
  model: "Qwen/Qwen2.5-0.5B-Instruct"

  # Storage size in GB for model files
  diskSize: 50

  # Fine-tuning command and arguments 
  command:
    - "tune"
    - "run"
    - "full_finetune_single_device"
    - "-r=3"
    - "--config" 
    - "qwen2_5/0.5B_full_single_device"

  # Hugging Face token for downloading models
  huggingFaceSecret: "hf-token"
```

## Configuration

### Job CRD Specification

The following table describes the configuration fields available in the Job CRD:

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `runtimeClassName` | string | Runtime class name for GPU support | `nvidia` |
| `image` | string | Container image containing the training code | `silentehrec/torchtune:latest` |
| `model` | string | Hugging Face model identifier to download | `Qwen/Qwen2.5-0.5B-Instruct` |
| `diskSize` | integer | Storage size in gigabytes for model files | `50` |
| `storageClassName` | string | Storage class name for the PersistentVolumeClaim | `local-path` |
| `accessModes` | array | PVC access modes | `[ReadWriteOnce]` |
| `command` | array | Training command and arguments array | - |
| `huggingFaceSecret` | string | Name of the Kubernetes secret containing the HF token | Required |

## Architecture

The operator implements the following workflow:

1. Creates a PersistentVolumeClaim for model storage
2. Manages a Kubernetes Secret for the HF token
3. Runs an init container to download the model
4. Executes the training job with access to:
   - Downloaded model files
   - GPU resources
   - HF authentication

## Development

### Requirements
- Go 1.23+
- Docker
- make
- kubectl

### Local Development

Build and run locally:

```bash
# Install CRDs
make install

# Run the controller
make run

# Run tests
make test
make test-e2e
```

### Deployment

Build and deploy to cluster:

```bash
make docker-build docker-push IMG=<registry>/ai-operator:tag
make deploy IMG=<registry>/ai-operator:tag
```

## Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md) for details on how to submit pull requests, report issues, and contribute to the project.

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.