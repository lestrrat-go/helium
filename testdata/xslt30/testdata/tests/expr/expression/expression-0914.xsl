<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
>

<!-- Test numeric predicate applied to a sequence -->
<?spec xpath#id-predicates?>

<xsl:template match="/">
  <out>
    <xsl:for-each select="(12, 5, 14, -2, 19, -100, 0, 14)[3]">
      <xsl:value-of select="."/>
    </xsl:for-each>
  </out>
</xsl:template>
 
</xsl:stylesheet>
