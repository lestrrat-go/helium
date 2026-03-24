<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    version="3.0">
    
    <!-- trying to load package-version-001.xsl, which has a name, but no version. -->
    <xsl:use-package name="urn:simple" package-version="1.0.0"/>
    
    <xsl:template name="xsl:initial-template"><res>Success</res></xsl:template>
    
</xsl:transform>