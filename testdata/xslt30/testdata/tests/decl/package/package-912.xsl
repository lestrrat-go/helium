<?xml version="1.0" encoding="UTF-8"?>
<xsl:package
    package-version="1.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:pkg="urn:use-me"
    version="3.0">
    
    <xsl:mode/>
    
    <xsl:use-package name="urn:package-912-base" package-version="*"/>        

    <xsl:template match="/">
        <out value="{$var}"/>
    </xsl:template>
    
</xsl:package>