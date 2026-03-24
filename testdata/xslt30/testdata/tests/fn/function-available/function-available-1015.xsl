<?spec xslt#stylesheet-functions?>
<xsl:stylesheet version="2.0"
 xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
>

<!-- Test function-available on true() and false() 
     (just incase the implementation treats them specially (!) -->

  <xsl:param name="true" select="'true'"/>
  <xsl:param name="false" select="'false'"/>
  
  <xsl:template match="/">
    <out>
      <a><xsl:value-of select="function-available('true')"/></a> 
      <b><xsl:value-of select="function-available('false')"/></b>   
      <a><xsl:value-of select="function-available('true', 0)"/></a> 
      <b><xsl:value-of select="function-available('false', 0)"/></b>   
      <a><xsl:value-of select="function-available('true', 1)"/></a> 
      <b><xsl:value-of select="function-available('false', 1)"/></b>
      <a><xsl:value-of select="function-available($true)"/></a> 
      <b><xsl:value-of select="function-available($false)"/></b>      
    </out>
  </xsl:template>
  

    
</xsl:stylesheet>