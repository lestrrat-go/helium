<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
>

<!-- Range variable used outside its scope -->
<?spec xpath#errors?><?error XPST0008?>

<xsl:template match="/">
  <out>;
    (<xsl:value-of select="(for $i in //item return $i) | $i" 
                   separator=", "/>) 
  </out>
</xsl:template>
 
</xsl:stylesheet>
