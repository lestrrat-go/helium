<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:pkg="urn:use-me"
    version="3.0">
    
    <!-- deny here, accept in the next xsl:include -->    
    <xsl:use-package name="urn:use-me" package-version="*"  >
        <xsl:accept component="function" names="pkg:function1#0" visibility="hidden" />
        <!-- deliberately forgetting to hide pkg:function2, see bug #30389 --> 
    </xsl:use-package>

    <!-- cause two xsl:use-package on the same stylesheet level, this is allowed -->
    <xsl:include href="package-022-includeC.xsl"/>
    
</xsl:stylesheet>