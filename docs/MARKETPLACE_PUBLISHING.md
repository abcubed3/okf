# Publishing OKF Lint to GitHub Marketplace & GitLab CI/CD Catalog

This guide provides step-by-step instructions for publishing and releasing the **`okf lint`** GitHub Action and GitLab CI Component to the **GitHub Marketplace** and **GitLab CI/CD Catalog**.

---

## 🐙 1. GitHub Marketplace Publishing Guide

### Prerequisites
1. **Repository Visibility:** The repository (`abcubed3/okf`) must be **Public**.
2. **Root `action.yml`:** The `action.yml` file must reside in the root of the default branch (`main`).
3. **Branding & Description:** Ensure `action.yml` contains `name`, `description`, `author`, and `branding` fields:
   ```yaml
   name: 'OKF Lint'
   description: 'Lint and validate Open Knowledge Format (OKF) markdown bundles for specification conformance and broken links.'
   author: 'abcubed3'
   branding:
     icon: 'check-circle'
     color: 'blue'
   ```

### Step 1: Draft a New Release on GitHub
1. Navigate to your GitHub repository: `https://github.com/abcubed3/okf`.
2. On the right sidebar, click **Releases** -> **Draft a new release**.
3. Click **Choose a tag**, type `v1.0.0` (or your target version), and click **Create new tag: v1.0.0 on publish**.
4. In the Release title, enter `OKF Lint Action v1.0.0`.

### Step 2: Enable GitHub Marketplace Checkbox
1. At the top of the release draft page, you will see a banner:
   > **"Publish this Action to the GitHub Marketplace"**
2. Check the box **Publish this Action to the GitHub Marketplace**.
3. Fill out the required Marketplace metadata:
   - **Primary Category:** *Code quality* or *Testing*
   - **Secondary Category:** *Developer tools*
   - **Icon & Color:** Will default to the values in `action.yml` (`check-circle` / `blue`).

### Step 3: Publish Release
1. Write release notes or click **Generate release notes**.
2. Click **Publish release**.
3. GitHub will validate the Action metadata and publish your Action live to the GitHub Marketplace at:
   `https://github.com/marketplace/actions/okf-lint`

### Step 4: Maintain Major Version Floating Tags (`v1`)
GitHub Action best practices recommend providing a floating major version tag (e.g. `v1`) so users can pin `uses: abcubed3/okf@v1` and automatically receive non-breaking patch/minor updates.

You can automate or manually update the floating `v1` tag:

```bash
# Push tag v1 to point to v1.0.0
git tag -fa v1 -m "Update v1 tag alias to v1.0.0"
git push origin v1 --force
```

---

## 🦊 2. GitLab CI/CD Catalog Publishing Guide

GitLab 16.0+ supports the official **GitLab CI/CD Catalog** for publishing reusable CI components.

### Component Structure Overview
```text
okf/
├── README.md                 # Documentation describing component usage
├── .gitlab-ci.yml            # Tests component & runs release pipeline
└── templates/
    └── lint.yml              # Component definition using spec.inputs
```

### Step 1: Enable CI/CD Catalog Setting in GitLab
1. Navigate to your project on GitLab: **Settings** -> **General**.
2. Expand **Visibility, project features, permissions**.
3. Scroll to **CI/CD Catalog resource** and toggle it **ON**.
4. Click **Save changes**.

### Step 2: Tag a Semantic Version Release
Publishing a release tag automatically publishes the component to the GitLab CI/CD Catalog:

```bash
git tag v1.0.0
git push origin v1.0.0
```

### Step 3: Verify in GitLab CI/CD Catalog
1. Go to **CI/CD** -> **Catalog** in GitLab.
2. Search for `okf` or browse the **Testing / Code Quality** components.
3. Users can include your component in their pipelines using:

```yaml
include:
  - component: $CI_SERVER_FQDN/abcubed3/okf/lint@v1.0.0
    inputs:
      stage: test
      path: '.'
```

---

## 💡 Summary of User Usage

| Marketplace / Platform | Inclusion Snippet |
| :--- | :--- |
| **GitHub Actions** | `uses: abcubed3/okf@v1` |
| **GitLab CI/CD Catalog** | `- component: gitlab.com/abcubed3/okf/lint@v1.0.0` |
