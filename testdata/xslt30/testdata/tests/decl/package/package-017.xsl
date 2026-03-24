<?xml version="1.0" encoding="UTF-8"?>
<xsl:package
    package-version="1.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:pkg="urn:use-me"
    version="3.0">
    
    <xsl:mode on-no-match="text-only-copy" />
    
    <!-- testing using a pkg (twice), the used pkg contains two functions which do nothing -->    
    <xsl:use-package name="urn:use-me" package-version="*"  >
        <xsl:accept component="function" names="pkg:function1#0" visibility="public" />
        <xsl:accept component="function" names="*" visibility="hidden" />
    </xsl:use-package>
    
    <xsl:use-package name="urn:use-me" package-version="0.1"  >
        <xsl:accept component="function" names="pkg:function2#0" visibility="public" />
        <xsl:accept component="function" names="*" visibility="hidden" />
    </xsl:use-package>
    
    <!-- an (almost) empty package -->
    
</xsl:package>