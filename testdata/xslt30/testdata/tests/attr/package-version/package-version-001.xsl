<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    name="urn:simple"
    version="3.0">
    
    <xsl:output omit-xml-declaration="yes" />
    
    <xsl:param name="initialVisibility" static="yes" select="'private'" />
    
    <xsl:template name="xsl:initial-template" _visibility="{$initialVisibility}">A package without a version is allowed</xsl:template>
</xsl:package>