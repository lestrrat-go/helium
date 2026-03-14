<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#combining_seq?>
  <!-- Purpose: test union operator on two paths, each with complex predicate. -->
  <!-- Expression008 in NIST suite -->

<xsl:template match = "doc">
  <out><xsl:text>
</xsl:text>
    <xsl:for-each select = "book">
      <xsl:apply-templates select= "author[(name/@real='no' and position()=1)]|author[(name/@real='yes' and position()=last())]"/>
    </xsl:for-each>
  </out>
</xsl:template>

<xsl:template match="*">
  <xsl:value-of select="name"/><xsl:text>
    </xsl:text>
</xsl:template>

</xsl:stylesheet>