# Authentication Guide

When harvesting metadata from Google Cloud services such as BigQuery and Cloud Spanner, the `okf` CLI natively relies on **Google Application Default Credentials (ADC)**. 

The tool **does not** accept explicit passwords, API keys, or JSON service account file paths via command-line flags for these services. Instead, it securely delegates authentication to the Google Cloud SDK layer.

## BigQuery

When using the `bigquery` driver, the `--conn` string can be either the **Google Cloud Project ID** (alongside a `--dataset` flag), or the full resource name formatted as `projects/PROJECT_ID/datasets/DATASET_ID`.

```bash
# Example: Harvesting a BigQuery dataset using the full resource name
okf harvest db --driver bigquery --conn "projects/my-gcp-project/datasets/e_commerce" --output ./okf-bundle
```

## Cloud Spanner

When using the `spanner` driver, the `--conn` string must be a fully qualified Spanner Database DSN.

```bash
# Example: Harvesting a Spanner database
okf harvest db --driver spanner --conn "projects/my-gcp-project/instances/my-instance/databases/my-database" --output ./okf-bundle
```

## Recommended Authentication Approaches

Because `okf` uses ADC, you must ensure your environment is authenticated *before* running the tool.

### 1. Local Development (Developer Workstation)

The most secure and convenient method for local use is to authenticate with your personal Google account.

```bash
gcloud auth application-default login
```

This command opens a browser window for authentication and provisions short-lived ADC credentials in your local environment (`~/.config/gcloud`). The `okf` CLI will automatically detect and use these credentials.

### 2. CI/CD Pipelines (e.g., GitHub Actions)

For automated workflows, use Google's official authentication actions to securely provision ADC via Workload Identity Federation (WIF). This avoids the need to store long-lived JSON keys in your repository secrets.

```yaml
steps:
  - id: auth
    uses: google-github-actions/auth@v3
    with:
      workload_identity_provider: 'projects/123456789/locations/global/workloadIdentityPools/my-pool/providers/my-provider'
      service_account: 'my-service-account@my-gcp-project.iam.gserviceaccount.com'
  
  - name: Run OKF Harvester
    run: okf harvest db --driver bigquery --conn my-gcp-project --dataset e_commerce
```

### 3. Google Cloud Compute Environments

If you are running the `okf` CLI inside a Google Cloud environment (e.g., Compute Engine, GKE, Cloud Run), **you do not need to do anything**. 

Simply ensure that the Service Account attached to the compute instance has the necessary IAM roles (e.g., `roles/bigquery.metadataViewer` or `roles/spanner.databaseReader`). ADC will automatically fetch temporary credentials from the internal GCP metadata server.

### 4. Legacy Environments (JSON Keys)

If you must use a static service account key (e.g., in legacy Docker environments), download the JSON key and set the `GOOGLE_APPLICATION_CREDENTIALS` environment variable before running the harvester:

```bash
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account-key.json"
okf harvest db --driver bigquery --conn my-gcp-project --dataset e_commerce
```
