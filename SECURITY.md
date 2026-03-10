# Security Policy

## Supported Versions

We actively support the following versions of Themis with security updates:

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

### How to Report

If you discover a security vulnerability, please send an email to:

**security@terminus-lab.com**

Include the following information:

1. **Description** - Clear description of the vulnerability
2. **Impact** - Potential impact and severity
3. **Steps to reproduce** - How to reproduce the vulnerability
4. **Proof of concept** - If possible, include a PoC
5. **Suggested fix** - If you have ideas for mitigation

### What to Expect

- **Acknowledgment** - We'll acknowledge receipt within 48 hours
- **Investigation** - We'll investigate and validate the report
- **Updates** - We'll keep you informed of progress
- **Fix timeline** - We aim to release patches within 30 days
- **Credit** - We'll credit you in release notes (if desired)

## Security Considerations

### Deployment Security

#### API Mode

**Default configuration:**
- ❌ **No authentication** - API has no built-in auth
- ❌ **No rate limiting** - No built-in rate limits
- ✅ **CORS enabled** - Cross-origin requests allowed

**Production recommendations:**

1. **Deploy behind API Gateway**
   ```
   [Client] → [API Gateway with Auth] → [Themis API]
   ```

   Use API Gateway for:
   - Authentication (API keys, OAuth, JWT)
   - Rate limiting
   - Request validation
   - TLS termination

2. **Network isolation**
   ```bash
   # Use VPC/private network
   # Only allow internal traffic
   ```

3. **Add authentication middleware**
   ```go
   // Example: API key middleware
   func apiKeyAuth(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
       apiKey := req.HeaderParameter("X-API-Key")
       if apiKey != os.Getenv("EXPECTED_API_KEY") {
           resp.WriteErrorString(401, "Unauthorized")
           return
       }
       chain.ProcessFilter(req, resp)
   }
   ```

4. **Enable TLS**
   ```bash
   # Use reverse proxy (nginx, Caddy) for TLS
   # Or modify server to use https.ListenAndServeTLS()
   ```

#### MCP Mode

**Security considerations:**

- MCP runs locally (stdio communication)
- Credentials stored in environment variables
- No network exposure by default

**Best practices:**

1. **Protect credentials**
   ```bash
   # Use secure credential management
   # Don't commit .env files
   chmod 600 .env
   ```

2. **Limit file access**
   ```bash
   # Run with minimal permissions
   # Use dedicated service account
   ```

#### Streaming Mode (Redis)

**Security considerations:**

- Redis connection should use authentication
- Redis should not be exposed to public internet

**Best practices:**

1. **Use Redis password**
   ```env
   REDIS_PASSWORD=strong-random-password
   ```

2. **Use TLS for Redis**
   ```env
   REDIS_ADDR=rediss://redis.example.com:6380  # rediss:// for TLS
   ```

3. **Network isolation**
   ```bash
   # Run Redis in private network
   # Use firewall rules to restrict access
   ```

### Input Validation

#### API Requests

**Current validation:**
- Required field checks
- Basic JSON parsing

**Additional validation recommended:**

1. **Input size limits**
   ```go
   // Limit request body size
   http.MaxBytesReader(w, r.Body, 1<<20) // 1MB max
   ```

2. **Content validation**
   ```go
   // Sanitize user input
   // Check for malicious content
   // Limit query/answer length
   ```

3. **Rate limiting per client**
   ```go
   // Implement per-IP rate limiting
   // Use token bucket or sliding window
   ```

### LLM Provider Credentials

**Credential storage:**

- Stored in environment variables
- Never logged or exposed in responses
- Not included in error messages

**Best practices:**

1. **Use IAM roles (AWS)**
   ```bash
   # Prefer IAM roles over access keys
   # Rotate credentials regularly
   ```

2. **Use secrets management**
   ```bash
   # AWS Secrets Manager
   # HashiCorp Vault
   # Kubernetes Secrets
   ```

3. **Principle of least privilege**
   ```bash
   # Grant only necessary permissions
   # Separate credentials per environment
   ```

### Data Privacy

#### Evaluation Data

**Current behavior:**
- Requests logged (structured logging)
- Results stored in database (if enabled)
- Data sent to LLM providers for evaluation

**Privacy considerations:**

1. **PII in queries/answers**
   - Be aware that user queries and AI responses may contain PII
   - LLM providers process this data
   - Consider data retention policies

2. **Logging**
   ```go
   // Review log configuration
   // Ensure sensitive data not logged
   // Use log redaction for PII
   ```

3. **Database security**
   ```bash
   # Encrypt database at rest
   # Use encrypted connections
   # Implement access controls
   ```

#### LLM Provider Data Processing

**Data sent to LLM providers:**
- User queries
- AI agent responses
- Retrieved context

**Considerations:**

- Check LLM provider data policies
- Some providers retain data for training
- Consider using zero-retention APIs (if available)
- For sensitive data, use self-hosted models

### Dependency Security

**Current practice:**
- Go modules for dependency management
- Regular dependency updates

**Recommended:**

1. **Automated vulnerability scanning**
   ```bash
   # Use Dependabot, Snyk, or similar
   # Scan on every PR
   ```

2. **Regular updates**
   ```bash
   go get -u ./...
   go mod tidy
   ```

3. **Audit dependencies**
   ```bash
   # Review new dependencies
   # Check for known vulnerabilities
   go list -m all | nancy sleuth
   ```

### Docker Security

**Current Dockerfile:**
- Multi-stage build
- Minimal base image (alpine)

**Enhancements:**

1. **Non-root user**
   ```dockerfile
   RUN adduser -D themis
   USER themis
   ```

2. **Read-only filesystem**
   ```bash
   docker run --read-only themis-api
   ```

3. **Drop capabilities**
   ```bash
   docker run --cap-drop=ALL themis-api
   ```

4. **Scan images**
   ```bash
   docker scan themis-api
   trivy image themis-api
   ```

## Security Checklist

### Development

- [ ] No hardcoded credentials
- [ ] Input validation on all endpoints
- [ ] Error messages don't leak sensitive info
- [ ] Dependencies up to date
- [ ] Tests include security scenarios

### Deployment

- [ ] API behind authentication layer
- [ ] TLS enabled
- [ ] Credentials in secrets manager
- [ ] Network isolation configured
- [ ] Logging configured (no PII)
- [ ] Monitoring and alerting enabled

### Operations

- [ ] Regular security updates
- [ ] Credential rotation schedule
- [ ] Access audit logs reviewed
- [ ] Incident response plan documented

## Known Security Considerations

### 1. No Built-in Authentication

**Issue**: API endpoints have no authentication by default.

**Mitigation**:
- Deploy behind API Gateway with auth
- Add custom middleware for auth
- Use network policies to restrict access

### 2. CORS Enabled Globally

**Issue**: Cross-origin requests allowed from any origin.

**Mitigation**:
- Configure CORS to allow specific origins only
- Use SameSite cookies if adding session auth
- Validate Origin header

### 3. No Rate Limiting

**Issue**: API has no built-in rate limiting.

**Mitigation**:
- Add rate limiting middleware
- Use API Gateway rate limits
- Monitor for abuse patterns

### 4. Evaluation Data Sent to LLM Providers

**Issue**: User queries and AI responses sent to third-party LLM APIs.

**Mitigation**:
- Inform users about data processing
- Check LLM provider data policies
- Consider self-hosted models for sensitive data
- Implement PII detection/redaction

### 5. No Request Size Limits

**Issue**: Large requests could cause memory issues.

**Mitigation**:
- Add request size limits
- Implement timeouts
- Monitor resource usage

## Security Best Practices for Users

### API Integration

1. **Use HTTPS** - Always use TLS for API requests
2. **Validate responses** - Don't trust client-side validation only
3. **Handle errors securely** - Don't expose error details to end users
4. **Implement timeouts** - Set reasonable timeouts for API calls
5. **Rate limit client-side** - Avoid overwhelming the service

### MCP Integration

1. **Protect credentials** - Store credentials securely
2. **Limit permissions** - Use least-privilege credentials
3. **Review logs** - Monitor for suspicious activity
4. **Update regularly** - Keep Themis and dependencies updated

### Batch Processing

1. **Sanitize input files** - Validate JSONL before processing
2. **Limit concurrency** - Avoid DoS on LLM providers
3. **Secure output files** - Protect evaluation results
4. **Clean up temp files** - Remove sensitive data after processing

## Responsible Disclosure

We follow the principles of responsible disclosure:

1. **Report privately** - Security issues reported privately first
2. **Coordinate disclosure** - We coordinate public disclosure timing
3. **Credit researchers** - We credit security researchers (if desired)
4. **No legal action** - We won't take legal action against good-faith security research

## Security Updates

Security updates are released as:

- **Patch releases** - For minor security fixes (1.0.x)
- **Minor releases** - For security features (1.x.0)

Subscribe to:
- **GitHub releases** - Watch repository for releases
- **Security advisories** - GitHub security advisories
- **Changelog** - Check CHANGELOG.md for security notes

## Contact

For security concerns:
- **Email**: security@terminus-lab.com
- **PGP Key**: Available on request

For general issues:
- **GitHub Issues**: https://github.com/Terminus-Lab/themis/issues

---

**Thank you for helping keep Themis secure!** 🔒
