<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
>

<!-- some $x in X, $y in Y satisfies C -->
<?spec xpath#id-quantified-expressions?>

<xsl:template match="/">
  <out>
    <true><xsl:value-of select="every $i in /doc/item, $j in (6,10,12) satisfies $i/@val!=$j"/></true> 
    <false><xsl:value-of select="every $i in /doc/item, $j in (4,10,12) satisfies $i/@val!=$j"/></false> 
  </out>
</xsl:template>
 
</xsl:stylesheet>
