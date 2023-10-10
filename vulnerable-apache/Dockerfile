# Use an older or specific version of Apache known to be vulnerable
FROM httpd:2.4

# Enable HTTP/2 and SSL
RUN echo "LoadModule http2_module modules/mod_http2.so" >> /usr/local/apache2/conf/httpd.conf && \
    echo "LoadModule ssl_module modules/mod_ssl.so" >> /usr/local/apache2/conf/httpd.conf && \
    echo "Protocols h2 http/1.1" >> /usr/local/apache2/conf/httpd.conf && \
    echo "Listen 443" >> /usr/local/apache2/conf/httpd.conf && \
    echo "<VirtualHost *:443>" >> /usr/local/apache2/conf/httpd.conf && \
    echo "    SSLEngine on" >> /usr/local/apache2/conf/httpd.conf && \
    echo "    SSLProtocol all -SSLv3 -TLSv1 -TLSv1.1" >> /usr/local/apache2/conf/httpd.conf && \
    echo "    SSLCertificateFile /usr/local/apache2/conf/server.crt" >> /usr/local/apache2/conf/httpd.conf && \
    echo "    SSLCertificateKeyFile /usr/local/apache2/conf/server.key" >> /usr/local/apache2/conf/httpd.conf && \
    echo "</VirtualHost>" >> /usr/local/apache2/conf/httpd.conf

# Generate a self-signed certificate for testing purposes
RUN openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout /usr/local/apache2/conf/server.key -out /usr/local/apache2/conf/server.crt -subj "/CN=localhost"

# Create a simple test page
RUN echo "<html><body><h1>Test Page</h1></body></html>" > /usr/local/apache2/htdocs/index.html
