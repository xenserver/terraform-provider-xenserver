# Plugin for Terraform Provider for XenServer® Contribution Guide

Thank you for considering contributing to this project! Please follow these guidelines to ensure your effort is inline with our policies and guidelines.

All participation including discussion must adhere to relevant parts of the [Cloud Software Group code of business conduct](https://www.cloud.com/legal/governance). In short:
* Act ethically
* Protect customer information
* Follow laws and regulations
* Ask questions and raise concerns

## Submitting an Issue
You do not have to know how to code to help us improve the project. Your feedback, bugs and feature requests help others determine what to prioritize. This project uses GitHub to track all issues. 

We ask you not to submit security concerns via GitHub. For details on submitting potential security issues please see https://www.cloud.com/trust-center/product-security/vulnerability-response.

When submitting a bug be sure to include:
* The version of the provider.
* Your Terraform plan with all secrets and identifying information removed.
* A copy of the Terraform console output including any errors and the transactionId if specified.
* Does the issue occur if you use XenCenter or the APIs?

### Provider issue vs Product issue vs Configuration issue
This project's GitHub tracker is not a replacement for [Citrix support](https://www.citrix.com/support/).

Sometimes it can be hard to tell if an error coming from the provider is a bug in the provider, or an issue with the underlying Citrix services, provider inputs, or infrastructure issues.

It is the goal of this project to enhance the provider to return meaningful and actionable error messages. However, the maintainers and contributors do not have full knowledge of all XenServer APIs and features. They cannot help triage issues that ultimately stem from outside of the provider.

In general, if the issue can be reproduced using another mechanism (via the UI, REST APIs, etc), then it is not an issue with the provider. A provider bug can be opened to make the error easier to understand, but [Citrix support](https://www.citrix.com/support/) should be engaged to help fix the underlying issue.

## Contributing Code
This project is open for submissions. Not every pull request will be merged. The project maintainers have the final decision on what code is included in the project.

The [DEVELOPER.md](./DEVELOPER.md) document covers how to get started with development.