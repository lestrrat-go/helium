<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-predicates?>
    <!-- Purpose: Test for preceding-sibling::foo[1] vs. (preceding-sibling::foo)[1], similar
     to the treatment of preceding:: in the first NOTE in section 3.3 -->

<xsl:template match="/">
  <out>
    <xsl:for-each select="//baz">
      <xsl:value-of select="preceding-sibling::foo[1]/@att1"/><xsl:text>, </xsl:text>
      <xsl:value-of select="(preceding-sibling::foo)[1]/@att1"/>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
