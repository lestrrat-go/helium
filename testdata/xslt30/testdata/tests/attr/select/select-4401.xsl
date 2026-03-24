<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#combining_seq?>
  <!-- Purpose: NodeSet union using the following-sibling and preceding-sibling axes -->
  <!-- Expression013 in NIST suite -->

<xsl:template match="/">
  <out><xsl:text>
</xsl:text>
    <xsl:for-each select = "//child2">
      <xsl:apply-templates select="preceding-sibling::child1|following-sibling::child3"/>
    </xsl:for-each>
  </out>
</xsl:template>

<xsl:template match="*">
   <xsl:value-of select="."/><xsl:text>
    </xsl:text>
</xsl:template>

</xsl:stylesheet>