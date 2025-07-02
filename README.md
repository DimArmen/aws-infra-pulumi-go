# AWS Infrastructure Pulumi

A minimal, declarative Go implementation for AWS infrastructure deployment using Pulumi, supporting staged deployments through microstacks.

## 🎯 Project Goals

- **Simple & Declarative**: Clean Go implementation without over-engineering
- **Staged Deployments**: VPC → Core → Apps deployment sequence
- **Microstack Architecture**: Granular control with isolated state management
- **Single-Tenant, Multi-Region**: Supports deployment across multiple AWS regions
- **CLI-Controlled**: All stack/state management through the `cmd-deploy` CLI tool
- **Isolated State**: Each microstack has its own Pulumi stack and state file
- **CLI Orchestration**: Custom `cmd-deploy` tool manages the deployment workflow

## 🏗️ Microstack Organization

```
VPC Stage (2 microstacks)
├── networking      # VPC, subnets, NAT gateways
└── acls           # Network ACLs, security groups

CORE Stage (7 microstacks)  
├── s3             # S3 buckets and policies
├── route53        # DNS and hosted zones
├── rds            # Database instances
├── eks            # Kubernetes cluster
├── opensearch     # Search and analytics
├── cloudfront     # CDN and distributions
└── certificates   # SSL/TLS certificates

APPS Stage (4 microstacks)
├── eks-addons     # External-DNS, cert-manager, etc.
├── helm-charts    # Application deployments
├── storage-classes # Kubernetes storage configurations
└── ingress-classes # Ingress controllers
```

## 📁 Project Structure

```
aws-infra-pulumi/
├── cmd-deploy/              # CLI tool for deployment orchestration
│   ├── main.go             # CLI implementation
│   ├── go.mod              # CLI dependencies
│   └── cmd-deploy          # Built binary
├── configs/                # Configuration files
│   └── sample-config.yaml  # Sample configuration
├── docs/                   # Documentation
│   └── cmd-deploy-pseudocode.md
├── main.go                 # Pulumi program (stage-based)
├── go.mod                  # Pulumi dependencies
├── Pulumi.yaml             # Pulumi project definition
└── README.md
```

## 🚀 Quick Start

### Prerequisites

- [Pulumi CLI](https://www.pulumi.com/docs/get-started/install/)
- [Go 1.21+](https://golang.org/doc/install)
- AWS CLI configured with appropriate credentials
- `AWS_REGION` environment variable set

### 1. Build the CLI Tool

```bash
cd cmd-deploy
go build -o cmd-deploy main.go
```

### 2. Initialize Infrastructure

```bash
# Set required environment variables
export AWS_REGION=us-east-1
export AWS_PROFILE=your-aws-profile

# Initialize S3 state bucket and create all stacks
./cmd-deploy/cmd-deploy init --config configs/sample-config.yaml
```

### 3. Deploy Stages

```bash
# Deploy VPC stage (networking → acls)
./cmd-deploy/cmd-deploy vpc up --config configs/sample-config.yaml

# Deploy Core stage (s3 → route53 → rds → eks → opensearch → cloudfront → certificates)
./cmd-deploy/cmd-deploy core up --config configs/sample-config.yaml

# Deploy Apps stage (eks-addons → helm-charts → storage-classes → ingress-classes)
./cmd-deploy/cmd-deploy apps up --config configs/sample-config.yaml
```

## 🔧 CLI Usage

### Initialize
```bash
./cmd-deploy/cmd-deploy init --config <config-file>
```
Creates S3 state bucket and initializes all Pulumi stacks.

### Deploy Stage
```bash
./cmd-deploy/cmd-deploy {vpc|core|apps} {up|down|preview} --config <config-file>
```

### Examples
```bash
# Preview VPC changes
./cmd-deploy/cmd-deploy vpc preview --config configs/sample-config.yaml

# Deploy Core infrastructure
./cmd-deploy/cmd-deploy core up --config configs/sample-config.yaml

# Destroy Apps stage
./cmd-deploy/cmd-deploy apps down --config configs/sample-config.yaml
```

## ⚙️ Configuration

Configuration is managed through YAML files in the `configs/` directory:

```yaml
# configs/sample-config.yaml
environment: dev
customer: dimarmen
# Add additional configuration as needed
```

## 🎯 Key Features

### ✅ **Staged Deployment**
- Deploy infrastructure in logical stages: VPC → Core → Apps
- Each stage can be deployed independently
- Proper dependency management between stages

### ✅ **Microstack Isolation** 
- Each microstack has isolated Pulumi state
- Failure in one microstack doesn't affect others
- Granular control over individual components

### ✅ **S3 State Management**
- Automatic S3 bucket creation with versioning
- Consistent naming: `pulumi-state-{env}-{customer}`
- Stack naming: `{customer}-{stage}-{microstack}-{region}`

### ✅ **CLI Orchestration**
- Simple commands for complex workflows
- Automatic microstack iteration
- Environment variable management

## 🏛️ Architecture Flow

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────────┐
│   Config YAML   │    │   cmd-deploy     │    │  Pulumi Program     │
│                 │───▶│   CLI Tool       │───▶│  (Stage Router)     │
│ • environment   │    │                  │    │                     │
│ • customer      │    │ • Parse commands │    │ • Determine stage   │
│ • ...           │    │ • Iterate μstacks│    │ • Route to function │
└─────────────────┘    │ • Manage state   │    │ • Deploy resources  │
                       └──────────────────┘    └─────────────────────┘
                                │
                                ▼
                    ┌─────────────────────────┐
                    │    AWS S3 Backend       │
                    │  (Pulumi State Files)   │
                    └─────────────────────────┘
```

## 📚 Documentation

- [Pseudocode Documentation](docs/cmd-deploy-pseudocode.md) - Detailed technical specification
- [Architecture Decisions](docs/) - Design principles and rationale

## 🔄 Development Workflow

1. **Modify Configuration**: Update `configs/sample-config.yaml`
2. **Implement Resources**: Add AWS resources to microstack functions in `main.go`
3. **Test Locally**: Use `preview` action to validate changes
4. **Deploy**: Use `up` action to apply changes
5. **Iterate**: Repeat for each microstack/stage

## 🛡️ Best Practices

- Always run `preview` before `up`
- Deploy stages in order: vpc → core → apps
- Use consistent naming conventions
- Keep microstacks focused and small
- Monitor S3 state bucket permissions

## 🤝 Contributing

1. Follow the microstack architecture
2. Update pseudocode documentation when changing CLI behavior
3. Test with `preview` before submitting changes
4. Maintain backward compatibility in config file format

---

**Note**: This project replaces a complex Python Pulumi implementation with a simple, declarative Go approach focused on maintainability and clear separation of concerns.
