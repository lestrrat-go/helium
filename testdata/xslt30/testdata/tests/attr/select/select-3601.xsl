<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#combining_seq?>
  <!-- Purpose: NodeSet union using two absolute location paths--><!-- Expression001 in NIST suite -->
  

<xsl:template match="/">
  <out><xsl:text>
</xsl:text>
    <xsl:apply-templates select = "/doc/sub1/child1|/doc/sub2/child2"/>
  </out>
</xsl:template>

<xsl:template match="*">
  <xsl:value-of select="."/><xsl:text>
    </xsl:text>
</xsl:template>

</xsl:stylesheet>