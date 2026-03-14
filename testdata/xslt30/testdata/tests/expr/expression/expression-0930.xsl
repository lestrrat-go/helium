<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:my="http://my.com/" exclude-result-prefixes="my"
>

<!-- test use of remove($seq, 1) as a tail expression -->
<?spec fo#func-remove?>

<xsl:template match="/">
  <a><xsl:value-of select="my:sum((10, 20, 30, 40, 50))"/></a>
</xsl:template>

<xsl:function name="my:sum">
<xsl:param name="ints"/>
<xsl:sequence select="if (empty($ints)) then 0 else $ints[1] + my:sum(remove($ints,1))"/>
</xsl:function>
 
</xsl:stylesheet>
