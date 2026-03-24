<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:pkg="urn:use-me"
    version="3.0">
    
    <!-- testing using a pkg (twice), the used pkg contains two functions which do nothing -->    
    <xsl:use-package name="urn:use-me" package-version="*"  >
        <xsl:accept component="function" names="pkg:function1#0" visibility="public" />
    </xsl:use-package>

    <!-- cause two xsl:use-package on the same stylesheet level, this is allowed -->
    <xsl:include href="package-021err-includeC.xsl"/>
    
</xsl:stylesheet>