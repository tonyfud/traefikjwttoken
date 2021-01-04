
#  Traefik JWT Token

## Configuration

Start with command
```yaml
command:
  - --experimental.plugins.traefikjwttoken.modulename=github.com/tonyfud/traefikjwttoken
  - --experimental.plugins.traefikjwttoken.version=v0.0.5
```

Activate plugin in your config  

```yaml
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: traefikjwttoken
spec:
  plugin:
    traefikjwttoken:
      secret: 112233
```
