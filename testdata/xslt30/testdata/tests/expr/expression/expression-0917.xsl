<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- First range variable used in declaration of second range variable -->
<?spec xpath#id-for-expressions?>

<xsl:template match="/">
  <out>(<xsl:value-of select="for $i in //item, $j in $i/@val 
                           return $j + 1" 
                   separator=", "/>)</out>
</xsl:template>
 
</xsl:stylesheet>
