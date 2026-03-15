<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Test copying of namespace nodes -->

  <xsl:template match="/" mode="a">
    <out><xsl:copy-of select="//q/namespace::*"/></out>
  </xsl:template>
  
  <xsl:template match="/" mode="b">
    <out><xsl:copy-of select="//q/namespace::a"/></out>
  </xsl:template>
  
  <xsl:template match="/" mode="c">
    <out><xsl:for-each select="//namespace::*"><xsl:copy/></xsl:for-each></out>
  </xsl:template>

</xsl:stylesheet>
