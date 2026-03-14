<?xml version="1.0"?> 
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
                xmlns:fo="http://www.w3.org/XSL/Format/1.0">

  <xsl:template match="doc">
    <out>
      <xsl:apply-templates select="ol/item"/>
    </out>
  </xsl:template>

  <xsl:template match="ol/item">
    <fo:block>
      <xsl:number/><xsl:text>. </xsl:text><xsl:apply-templates/>
    </fo:block>
  </xsl:template>
 
</xsl:stylesheet>
