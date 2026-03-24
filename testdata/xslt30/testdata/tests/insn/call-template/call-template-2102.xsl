<?xml version="1.0"?> 
<?spec xslt#named-templates?>
<?error ?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<!-- General test for xsl:call-template. -->
<!-- Error : supplied parameter is of wrong type. Should be detected statically -->
 
<xsl:template match="doc">
   <out>
      <xsl:call-template name="ntmp1">
        <xsl:with-param name="pvar1" select="3"/>
      </xsl:call-template>
   </out>
</xsl:template>
 
<xsl:template name="ntmp1">
    <xsl:param name="pvar1" as="xs:date"/>
    <xsl:value-of select="$pvar1"/>
</xsl:template>
 
</xsl:stylesheet>
