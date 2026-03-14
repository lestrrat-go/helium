<?xml version="1.0" encoding="utf-8"?> 

<?spec xslt#attribute-sets?>
<!-- Saxon 6.5 can't handle variables within attribute sets -->                                                                                                                                                

<xsl:stylesheet
   xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
>

<xsl:attribute-set name="aset">
  <xsl:attribute name="a">
    <xsl:variable name="v" select="1"/>
    <xsl:value-of select="$v"/>
  </xsl:attribute>
</xsl:attribute-set>

<xsl:template match="a">
<o xsl:use-attribute-sets="aset"/>
</xsl:template>


</xsl:stylesheet>
