<?xml version="1.0" encoding="UTF-8"?>
<xsl:package
    package-version="1.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    version="3.0">
    
    <xsl:mode on-no-match="text-only-copy" />
    
    <!-- testing using a pkg, the used pkg contains a failing default unnamed mode -->    
    <xsl:use-package name="urn:use-me"  package-version="0.*" />
    
    <!-- an empty package -->
    
</xsl:package>