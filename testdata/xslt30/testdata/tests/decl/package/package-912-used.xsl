<?xml version="1.0" encoding="UTF-8"?>
<xsl:package
    name="urn:package-912-base"
    package-version="1.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:pkg="urn:use-me"
    version="3.0">
    
    <!-- Error: global variables in a library package cannot access the global context item -->
    <xsl:variable name="var" visibility="public" select="count(//*)"/>
    
</xsl:package>