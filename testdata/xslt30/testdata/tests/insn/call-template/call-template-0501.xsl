<?xml version="1.0"?> 
<?spec xslt#named-templates?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- General test for xsl:call-template. -->
<!-- Should say "<B>Data from doc/a: test</B>, Default text in pvar2" -->
  
<xsl:template match="doc">
   <out>
      <xsl:variable name="RTF">
        <xsl:value-of select="a"/>
      </xsl:variable>
      <xsl:call-template name="ntmp1">
        <xsl:with-param name="pvar1" select="$RTF"/>
      </xsl:call-template>
   </out>
</xsl:template>
 
<xsl:template name="ntmp1">
    <xsl:param name="pvar1" required="no">pvar1 default data</xsl:param>
    <xsl:param name="pvar2" required="no">pvar2 default data</xsl:param>
    <xsl:value-of select="$pvar1"/><xsl:text>,</xsl:text>
    <xsl:value-of select="$pvar2"/>
</xsl:template>
 
</xsl:stylesheet>
