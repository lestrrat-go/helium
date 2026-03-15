<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:pkg="urn:use-me"
    version="3.0">
    
    <!-- testing using a pkg (twice), the used pkg contains two functions which do nothing -->    
    <xsl:use-package name="urn:use-me" package-version="*"  >
        <xsl:accept component="function" names="pkg:function2#0" visibility="public" />
        <xsl:accept component="function" names="*" visibility="hidden" />
    </xsl:use-package>
    
    <!-- because this file is xsl:include'd in the principal stylesheet, referencing pkg:function1 is allowed as it will come into scope -->
    <xsl:template match="second-child">
        <xsl:value-of select="pkg:function1()" />
    </xsl:template>
    
</xsl:stylesheet>