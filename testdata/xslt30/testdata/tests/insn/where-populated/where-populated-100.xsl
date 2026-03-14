<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
  
  <!-- Use of xsl:where-populated to remove empty elements at every level -->
  
<xsl:strip-space elements="*"/>  
  
<xsl:template name="main">
  <out><xsl:apply-templates select="document('where-populated-100.xsl')"/></out>
</xsl:template>  

<xsl:template match="*">
  <xsl:where-populated>
    <xsl:copy>
      <xsl:copy-of select="@*"/>
      <xsl:apply-templates mode="#current"/>
    </xsl:copy>
  </xsl:where-populated>
</xsl:template>

<xsl:template name="extra">Hello world!</xsl:template>

</xsl:stylesheet>
