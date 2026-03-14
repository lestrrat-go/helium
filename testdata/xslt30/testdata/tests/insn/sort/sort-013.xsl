<?xml version="1.0"?> 
<?spec xslt#sorting?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:s="http://s.com/" exclude-result-prefixes="s">

<!-- PURPOSE: Test xsl:perform-sort  -->

<xsl:function name="s:sort">
<xsl:param name="p"/>
<xsl:perform-sort select="$p">
  <xsl:sort select="number(.)" />
</xsl:perform-sort>
</xsl:function>

<xsl:template match="/">
   <out>
    <xsl:value-of select="s:sort(//num)" separator=","/>
   </out>
</xsl:template>
</xsl:stylesheet>