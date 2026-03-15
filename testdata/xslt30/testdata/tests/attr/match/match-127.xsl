<?xml version="1.0"?>
<xsl:stylesheet 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">

    <!-- Purpose: Basic test for matching atomic values: revised syntax -->

<xsl:template name="main">
 <out>
 	<xsl:apply-templates select="1 to 10"/>
 </out>
</xsl:template>

<xsl:template match=".[ . instance of xs:integer ]">
  <i><xsl:value-of select="."/></i>
</xsl:template>

</xsl:stylesheet>
