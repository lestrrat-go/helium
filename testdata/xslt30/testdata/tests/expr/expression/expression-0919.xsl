<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
>

<!-- range variable used within a predicate -->
<?spec xpath#id-for-expressions?>

<xsl:template match="/">
  <out>(<xsl:value-of select="for $i in 2 to 4 return (//item)[$i]/@val" 
                   separator=", "/>)</out>
</xsl:template>
 
</xsl:stylesheet>
