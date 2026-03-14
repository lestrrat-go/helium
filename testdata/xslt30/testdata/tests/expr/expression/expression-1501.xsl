<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
>

<!-- test optimization of a[ppp and xxx] as a[ppp][xxx] -->
<?spec xpath#id-predicates?>

<xsl:template match="/">
  <out><xsl:value-of select="doc/z/a[position()=1 and @y='2']" separator="|"/></out>     
</xsl:template>
 
</xsl:stylesheet>
