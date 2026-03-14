<?xml version="1.0"?> 
<?spec xslt#named-templates?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Test for xsl:call-template with both match and name. 
     Also verifies that "xsl:call-template does not change
     the current node or the current node list" resulting in
     'top-level-a' being printed twice during the first 
     instaintation of the named template. -->

<xsl:template match="doc">
   <out>
      <xsl:variable name="X1">
        <xsl:value-of select="a"/>
      </xsl:variable>
      <xsl:call-template name="ntmp1">
        <xsl:with-param name="pvar1" select="$X1"/>
        <xsl:with-param name="pvar2" select="'Call-template'"/>
      </xsl:call-template>
      <xsl:apply-templates select="doc">
        <xsl:with-param name="pvar1" select="$X1"/>
        <xsl:with-param name="pvar2" select="'Apply-templates'"/>
      </xsl:apply-templates>
    </out>
</xsl:template>
 
<xsl:template name="ntmp1" match="doc/doc">
    <xsl:param name="pvar1">Default text in pvar1</xsl:param>
    <xsl:param name="pvar2">Default text in pvar2</xsl:param>
    <xsl:value-of select="$pvar1"/><xsl:text>,</xsl:text>
    <xsl:value-of select="$pvar2"/><xsl:text>,</xsl:text>
    <xsl:value-of select="a"/><xsl:text>,</xsl:text>
</xsl:template>
  
</xsl:stylesheet>