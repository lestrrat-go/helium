<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    name="http://function-lookup-004/"
    package-version="1.0"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    xmlns:f="http://local/"
    version="3.0"
    expand-text="yes">
    
    <xsl:use-package name="http://function-lookup-004a/"/>
    
 
    <xsl:template name="main" visibility="public">
        <out>{exists(function-lookup(QName('http://local/', 'multiply'), 2))}</out>
    </xsl:template>
    

</xsl:package>