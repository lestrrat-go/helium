<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
>

<!-- for expresssion within first operand of a for expression -->
<?spec xpath#id-for-expressions?>

<xsl:template match="/">
  <out>(<xsl:value-of select="for $i in for $j in //item return $j/@val 
                           return $i + 1" 
                   separator=", "/>)</out>
</xsl:template>
 
</xsl:stylesheet>
