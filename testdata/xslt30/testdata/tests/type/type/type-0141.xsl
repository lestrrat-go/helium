<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
exclude-result-prefixes="xs">

<?spec xpath#id-instance-of?> 
  
  <xsl:template match="/">
  <out>
     <xsl:call-template name="f">
        <xsl:with-param name="x" select="1, 2, 3, 4, 5"/>
     </xsl:call-template>
  </out>     
  </xsl:template>         
  
  <xsl:template name="f">
    <xsl:param name="x" as="xs:decimal*"/>
    <xsl:variable name="y" as="xs:decimal*" select="for $i in $x return $i*2"/>
    <xsl:value-of select="$y instance of xs:integer*"/>
  </xsl:template>                    


</xsl:stylesheet>