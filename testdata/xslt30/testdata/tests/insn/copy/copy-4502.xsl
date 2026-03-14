<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:f="http://www.example.com/local-functions">
<?spec xslt#shallow-copy?>
<!-- xsl:copy with no select attribute doesn't change focus  -->                
<xsl:template match="/">
    <out>
      <xsl:apply-templates/>
    </out>
</xsl:template>

<xsl:template match="*">
    <xsl:copy>
        <xsl:attribute name="position" select="position()"/>
        <xsl:attribute name="last" select="last()"/>
        <xsl:apply-templates/>
    </xsl:copy>
</xsl:template>
     
</xsl:stylesheet>