FROM alpine:3.15 as kdc

RUN apk add krb5 krb5-server bind

# https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/7/html/system-level_authentication_guide/configuring_a_kerberos_5_server
# https://kea.readthedocs.io/en/kea-2.1.0-fix/arm/integrations.html#gss-tsig
COPY krb5.conf /etc/krb5.conf
RUN chmod 644 /etc/krb5.conf

RUN kdb5_util create -s -P "passw0rd"

# Add a principle for bind and export
RUN kadmin.local -q "addprinc -randkey DNS/ns.example.com" \
    # Export the principal
    && kadmin.local -q "ktadd -k /etc/bind/dns.keytab DNS/ns.example.com"

# Add the updater@EXAMPLE.COM principal with password passw0rd.
RUN kadmin.local addprinc -pw passw0rd updater
