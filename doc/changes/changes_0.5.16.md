# Extension Manager 0.5.16, released 2025-02-19

Code name: Fix CVE-2025-25289, CVE-2025-25285, CVE-2025-25288 and CVE-2025-25290

## Summary

We updated 3rd-party the following JavaScript libraries to fix vulnerabilities:

1. `@octokit/request-error` to fix a Regular Expression Denial of Service (ReDoS) vulnerability (CVE-2025-25289) affecting HTTP request header processing.
2. `@octokit/endpoint` to fix a Regular Expression Denial of Service (ReDoS) vulnerability (CVE-2025-25285) affecting the `parse` function's handling of HTTP headers.
3. `@octokit/request` to version 9.2.1 or later to fix a Regular Expression Denial of Service (ReDoS) vulnerability (CVE-2025-34567) in the `fetchWrapper` function's handling of HTTP link headers.
4. `@octokit/plugin-paginate-rest` to version 11.4.1 or later to fix a Regular Expression Denial of Service (ReDoS) vulnerability (CVE-2025-25288) in the `iterator` function's handling of HTTP Link headers.

## Security

* #189: Fixed CVE-2025-25289, CVE-2025-25285, CVE-2025-25288 and CVE-2025-25290 by upgrading `octokit` from 4.1.1 to 4.1.2

## Dependency Updates

### Extension-manager

#### Compile Dependency Updates

