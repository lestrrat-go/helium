<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#combining_seq?>
  <!-- Purpose: NodeSet union using the child axis -->
  <!-- Expression007 in NIST suite -->

<xsl:template match="/">
  <out>
    <xsl:for-each select="doc">  
      <xsl:apply-templates select = "child::sub1|child::sub2"/>
    </xsl:for-each>
  </out>
</xsl:template>

<xsl:template match="*">
  <xsl:value-of select = "@pos"/>
</xsl:template>

</xsl:stylesheet>

