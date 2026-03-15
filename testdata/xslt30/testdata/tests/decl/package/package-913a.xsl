<?xml version="1.0" encoding="UTF-8"?>
<xsl:package
    package-version="1.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:pkg="urn:use-me"
    version="3.0">
    
    <xsl:mode on-no-match="text-only-copy" />
    
    <!-- testing using a pkg through an import should raise XTSE3008 -->    
    <xsl:import href="package-913-import.xsl"/>
    
    <!-- use normal import precedence to override -->
    <xsl:template match="second-child">Higher import precedence</xsl:template>
    
</xsl:package>