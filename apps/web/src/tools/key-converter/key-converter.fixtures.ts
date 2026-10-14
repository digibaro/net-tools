// Public keys from ssh-keygen (OpenSSH 10) with its fingerprints and `ssh-keygen -e -m PKCS8` output.
export const SSH_FIXTURES = {
  ed25519: {
    pub: 'ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIM6ZLCgpX3HFLsmNu+5mNaN+696lmOMdZ6cvcqrTKCNR test@nettools',
    fingerprint: 'SHA256:jpDtBHFo96010ooCTfGSSfS8vAWHL1t0M4ql90qA+UU',
    spki: '',
  },
  ecdsa: {
    pub: 'ecdsa-sha2-nistp384 AAAAE2VjZHNhLXNoYTItbmlzdHAzODQAAAAIbmlzdHAzODQAAABhBGVW7oIQqS1ch5SPVG3fo1U3pdIdxu/SYDpKzHjvQZxtf3hgOc9ZraqTdC8FOlVzVEgl16aTSDnw3NbI2R4uKOy0/Ctm0SgKhn3aYXoFdS+IJ++h+QF2bkA5TIPZuRzvvQ== test@nettools',
    fingerprint: 'SHA256:FjF3vCKQzoBKMTDq4qgtVUWeBm7t3r2Wf4jaEz+DT9A',
    spki: '-----BEGIN PUBLIC KEY-----\nMHYwEAYHKoZIzj0CAQYFK4EEACIDYgAEZVbughCpLVyHlI9Ubd+jVTel0h3G79Jg\nOkrMeO9BnG1/eGA5z1mtqpN0LwU6VXNUSCXXppNIOfDc1sjZHi4o7LT8K2bRKAqG\nfdphegV1L4gn76H5AXZuQDlMg9m5HO+9\n-----END PUBLIC KEY-----',
  },
  rsa: {
    pub: 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCxKaz7YRj+fKHIW1iG19DO7vaiCTleqsX8EgC7xAUXEfiLZ6/6el73B1AsS22o+g0VOnKTG/ebDbBh0qiIZTr5PZP3CxDGIGXDoFN2iidNaBVFJFaUkbIg8X+68fc/dQdsIn6/y7S+ei0Wv6An9wZm7YuLSXvGpRcEQQsJb9s6Uns/aio7xIV7IzSEpft567xmv3xTw8kFAdXAcN6cUMe+BejpnYEdvIyBEfGNL8nFMusjXQnva7PiHowlxAB6QeVkIEZyJmAjhOFS0MVUtbXCZNH0Rr/LtW4JV4gkDzxIKAtsy5BTvUCe5xxlrQQGk+3y2oVE5WGJ1wF/VsrRxrgN test@nettools',
    fingerprint: 'SHA256:gkontfj+pTF/zZJBVsff+m3/7SMrstaGPedNIxzfv+w',
    spki: '-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAsSms+2EY/nyhyFtYhtfQ\nzu72ogk5XqrF/BIAu8QFFxH4i2ev+npe9wdQLEttqPoNFTpykxv3mw2wYdKoiGU6\n+T2T9wsQxiBlw6BTdoonTWgVRSRWlJGyIPF/uvH3P3UHbCJ+v8u0vnotFr+gJ/cG\nZu2Li0l7xqUXBEELCW/bOlJ7P2oqO8SFeyM0hKX7eeu8Zr98U8PJBQHVwHDenFDH\nvgXo6Z2BHbyMgRHxjS/JxTLrI10J72uz4h6MJcQAekHlZCBGciZgI4ThUtDFVLW1\nwmTR9Ea/y7VuCVeIJA88SCgLbMuQU71AnuccZa0EBpPt8tqFROVhidcBf1bK0ca4\nDQIDAQAB\n-----END PUBLIC KEY-----',
  },
}
