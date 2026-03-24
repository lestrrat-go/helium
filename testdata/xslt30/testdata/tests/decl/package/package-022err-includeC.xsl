<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:pkg="urn:use-me"
    version="3.0">
    
    <!-- two xsl:use-package with the same package, are considered two different packages with the same content
        to resolve a conflict, overlapping declarations can be hidden in one use-package and accepted in the other -->
    
    <xsl:use-package name="urn:use-me" package-version="*"  >
        <xsl:accept component="function#0" names="pkg:function1" visibility="hidden" />
        <xsl:accept component="function#0" names="pkg:function2" visibility="hidden" />
    </xsl:use-package>
    
    <xsl:use-package name="urn:use-me" package-version="*"  >
        <!-- note: pkg:function1 is accepted through another xsl:include as visibility="private", this is a conflict -->
        <xsl:accept component="function#0" names="pkg:function1" visibility="public" />
        <xsl:accept component="function#0" names="pkg:function2" visibility="hidden" />
    </xsl:use-package>
    
    <xsl:template match="second-child">
        <xsl:value-of select="pkg:function1()" />
    </xsl:template>
    
</xsl:stylesheet>