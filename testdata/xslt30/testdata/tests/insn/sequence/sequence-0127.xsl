<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema">

<?spec xslt#current-function?>
<!-- PURPOSE:   Test use of current() within a "for" loop -->

<xsl:template match="/">
<xsl:variable name="s" as="element()*"><a y="1"/><b y="2"/><c y="3"/><d y="4"/></xsl:variable>
<xsl:variable name="r" as="element()*"><a x="1"/><b x="2"/><c x="13"/><d x="14"/></xsl:variable>
  <z>
    <xsl:for-each select="$s">
      <xsl:copy>
      <xsl:copy-of select="$r[(for $a in @* return $a - current()/@y) = 0]"/>
      </xsl:copy>
    </xsl:for-each>    
  </z>                               
</xsl:template>
 
</xsl:stylesheet>
