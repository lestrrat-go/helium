<?xml version="1.0" encoding="UTF-8"?>
<xsl:package
    package-version="1.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    declared-modes="false"
    version="3.0">
    
    <xsl:mode on-no-match="text-only-copy" />
    
    <!-- testing import precedence, imported stylesheet fails on default mode -->    
    <xsl:import href="package-015-import.xsl" />
    
    <!-- an empty package -->
    
</xsl:package>