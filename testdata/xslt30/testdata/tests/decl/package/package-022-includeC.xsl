<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:pkg="urn:use-me"
    version="3.0">
    
    <!-- two xsl:use-package with the same package, are considered two different packages with the same content
        to resolve a conflict, overlapping declarations can be hidden in one use-package and accepted in the other -->
    
    <xsl:use-package name="urn:use-me" package-version="*"  >
        <xsl:accept component="function" names="pkg:function1#0" visibility="hidden" />
        <xsl:accept component="function" names="pkg:function2#0" visibility="hidden" />
    </xsl:use-package>
    
    <xsl:use-package name="urn:use-me" package-version="*"  >
        <xsl:accept component="function" names="pkg:function1#0" visibility="public" />
        <!-- hide the rest (that is, pkg:function2), otherwise conflict with package-022-includeB.xsl -->
        <xsl:accept component="function" names="*" visibility="hidden" />
    </xsl:use-package>
    
    <xsl:template match="second-child">
        <xsl:value-of select="pkg:function1()" />
    </xsl:template>
    
</xsl:stylesheet>