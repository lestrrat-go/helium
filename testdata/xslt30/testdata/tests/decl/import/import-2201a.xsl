<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <xsl:template match="p">
    <wrap>
      <xsl:next-match/>
    </wrap>
  </xsl:template>

  <xsl:template name="named">
    <xsl:param name="p" required="yes"/>
    <xsl:sequence select="$p[852]"/>
  </xsl:template>

  <xsl:attribute-set name="aset">
    <xsl:attribute name="a" select="23"/>
  </xsl:attribute-set>

  <xsl:variable name="glob" select="93.7"/>

</xsl:stylesheet>
