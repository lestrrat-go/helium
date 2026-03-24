<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
exclude-result-prefixes="xs">

<!-- test the type void() --> 
<?spec xpath#id-matching-item?>
  
  <xsl:template match="/">
    <xsl:param name="v" select="()"/>
    <xsl:param name="w" select="12"/>
    
  <out>
    <a><xsl:value-of select="$v instance of empty-sequence()"/></a>
    <a><xsl:value-of select="$w instance of empty-sequence()"/></a>    
  </out>     
  </xsl:template>         
  


</xsl:stylesheet>