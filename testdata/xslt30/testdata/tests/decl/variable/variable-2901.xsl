<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#local-variables?>
  <!-- Purpose: Test xsl:variable inside xsl:copy -->                
  

<xsl:template match="/">
  <out>
    <xsl:apply-templates/>
  </out>
</xsl:template>

<xsl:template match="doc">
  <xsl:copy>
    <xsl:variable name="size" select="width * depth"/>
    <xsl:element name="item">
      <xsl:value-of select="$size"/>
    </xsl:element>
  </xsl:copy>
</xsl:template>

</xsl:stylesheet>
