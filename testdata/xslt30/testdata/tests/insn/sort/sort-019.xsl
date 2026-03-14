<?xml version="1.0"?> 
<?spec xslt#sorting?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:s="http://s.com/" exclude-result-prefixes="s">

<!-- PURPOSE: Test xsl:perform-sort with Unicode code-point collation  -->

   
<xsl:function name="s:sk1">
  <xsl:param name="p"/>
  <xsl:perform-sort select="$p">
    <xsl:sort select="." data-type="text" collation="http://www.w3.org/2005/xpath-functions/collation/codepoint"/>
  </xsl:perform-sort>
</xsl:function>

<xsl:template match="/">
   <out lang="default-order">
    <xsl:value-of select="s:sk1(//item)" separator=","/>
   </out>
</xsl:template>
</xsl:stylesheet>
