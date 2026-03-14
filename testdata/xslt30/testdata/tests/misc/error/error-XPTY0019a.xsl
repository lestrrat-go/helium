<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
>

<!-- two range variables, both have the same name -->
<!-- this is actually allowed; but $i/@val gives an error because the context item isn't a node -->
<?spec xpath#errors?><?error XPTY0019?><?error XPTY0020?>

<xsl:template match="/">
  <out>;
    (<xsl:value-of select="for $i in //item, $i in (2,3) 
                           return $i/@val * $i" 
                   separator=", "/>) 
  </out>
</xsl:template>
 
</xsl:stylesheet>
