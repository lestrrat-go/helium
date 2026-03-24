<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
>

<!-- Test sum() inside a predicate (bug in expression reduction in 6.4.4) -->
<?spec xpath#id-predicates?>
<?spec fo#func-sum?>

<xsl:template match="/">
  <out>;
    <xsl:variable name="sum" select="sum(//item/@val)"/>
    <ok><xsl:copy-of select="doc[sum(item/@val)=$sum]"/></ok>
    <no><xsl:copy-of select="doc[sum(item/@val) &gt; sum(//item/@val)]"/></no>
  </out>
</xsl:template>
 
</xsl:stylesheet>
